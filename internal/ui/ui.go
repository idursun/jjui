package ui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	ui "github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/bookmarks"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/git"
	"github.com/idursun/jjui/internal/ui/helppage"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/internal/ui/undo"

	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/status"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	revisions      *revisions.Model
	revsetModel    revset.Model
	previewModel   tea.Model
	previewVisible bool
	diff           tea.Model
	state          common.State
	error          error
	status         *status.Model
	output         string
	width          int
	height         int
	context        context.AppContext
	keyMap         config.KeyMappings[key.Binding]
	stacked        tea.Model
}

func (m Model) Init() tea.Cmd {
	return tea.Sequence(tea.SetWindowTitle("jjui"), m.revisions.Init())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(common.CloseViewMsg); ok && (m.diff != nil || m.stacked != nil) {
		m.diff = nil
		m.stacked = nil
		return m, nil
	}

	var cmd tea.Cmd
	if m.diff != nil {
		m.diff, cmd = m.diff.Update(msg)
		return m, cmd
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.revsetModel.Editing {
			m.revsetModel, cmd = m.revsetModel.Update(msg)
			m.state = common.Loading
			return m, cmd
		}

		if m.revisions.IsFocused() {
			m.revisions, cmd = m.revisions.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		if r, ok := m.previewModel.(common.Focusable); ok && r.IsFocused() {
			m.previewModel, cmd = m.previewModel.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		if m.stacked != nil {
			m.stacked, cmd = m.stacked.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		switch {
		case key.Matches(msg, m.keyMap.Cancel) && m.state == common.Error:
			m.state = common.Ready
			m.error = nil
		case key.Matches(msg, m.keyMap.Cancel) && m.stacked != nil:
			m.stacked = nil
		case key.Matches(msg, m.keyMap.Revset) && m.revisions.InNormalMode():
			m.revsetModel, _ = m.revsetModel.Update(revset.EditRevSetMsg{Clear: m.state != common.Error})
		case key.Matches(msg, m.keyMap.Git.Mode) && m.revisions.InNormalMode():
			m.stacked = git.NewModel(m.context, m.revisions.SelectedRevision(), m.width, m.height)
		case key.Matches(msg, m.keyMap.Undo) && m.revisions.InNormalMode():
			m.stacked = undo.NewModel(m.context)
			cmds = append(cmds, m.stacked.Init())
		case key.Matches(msg, m.keyMap.Bookmark.Mode) && m.revisions.InNormalMode():
			m.stacked = bookmarks.NewModel(m.context, m.revisions.SelectedRevision(), m.width, m.height)
			cmds = append(cmds, m.stacked.Init())
		case key.Matches(msg, m.keyMap.Help):
			cmds = append(cmds, common.ToggleHelp)
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Mode):
			m.previewVisible = !m.previewVisible
			cmds = append(cmds, common.SelectionChanged)
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.ToggleFocus):
			if !m.previewVisible {
				cmds = append(cmds, common.SelectionChanged)
			}
			m.previewVisible = true
			cmds = append(cmds, preview.Focus)
			return m, tea.Batch(cmds...)
		}
	case common.ToggleHelpMsg:
		if m.stacked == nil {
			m.stacked = helppage.New(m.context)
			if p, ok := m.stacked.(common.Sizable); ok {
				p.SetHeight(m.height - 2)
				p.SetWidth(m.width)
			}
		} else {
			m.stacked = nil
		}
		return m, nil
	case common.ShowDiffMsg:
		m.diff = diff.New(string(msg), m.width, m.height)
		cmds = append(cmds, m.diff.Init())
		return m, tea.Batch(cmds...)
	case common.CommandCompletedMsg:
		m.output = msg.Output
	case common.UpdateRevisionsFailedMsg:
		m.state = common.Error
		m.output = msg.Output
		m.error = msg.Err
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.previewVisible {
			m.revisions.SetWidth(m.width / 2)
		} else {
			m.revisions.SetWidth(m.width)
		}
		m.revisions.SetHeight(m.height - 4)
		if p, ok := m.previewModel.(common.Sizable); ok && m.previewVisible {
			p.SetWidth(m.width / 2)
			p.SetHeight(m.height - 4)
		}
		if s, ok := m.stacked.(common.Sizable); ok {
			s.SetWidth(m.width - 2)
			s.SetHeight(m.height - 2)
		}
		m.status.SetWidth(m.width)
	}

	m.status, cmd = m.status.Update(msg)
	cmds = append(cmds, cmd)

	if m.stacked != nil {
		m.stacked, cmd = m.stacked.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.revisions, cmd = m.revisions.Update(msg)
	cmds = append(cmds, cmd)

	if m.previewVisible {
		m.previewModel, cmd = m.previewModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.diff != nil {
		return m.diff.View()
	}

	topView := m.revsetModel.View()
	if m.state == common.Error {
		topView += fmt.Sprintf("\n%s\n", m.output)
	}
	topViewHeight := lipgloss.Height(topView)

	m.status.SetCurrentOperation(m.revisions.CurrentOperation())
	footer := m.status.View()
	footerHeight := lipgloss.Height(footer)

	m.revisions.SetWidth(m.width)
	if m.previewVisible {
		m.revisions.SetWidth(m.width / 2)
		if p, ok := m.previewModel.(common.Focusable); ok && p.IsFocused() {
			m.revisions.SetWidth(4)
		}
	}
	m.revisions.SetHeight(m.height - footerHeight - topViewHeight)
	revisionsView := m.revisions.View()

	previewView := ""
	if p, ok := m.previewModel.(common.Sizable); ok && m.previewVisible {
		p.SetWidth(m.width - lipgloss.Width(revisionsView))
		p.SetHeight(m.height - footerHeight - topViewHeight)
		previewView = m.previewModel.View()
	}

	centerView := lipgloss.JoinHorizontal(lipgloss.Left, revisionsView, previewView)

	if m.stacked != nil {
		stackedView := m.stacked.View()
		sx := (m.width - lipgloss.Width(stackedView)) / 2
		sy := (m.height - lipgloss.Height(stackedView)) / 2
		centerView = ui.Stacked(centerView, stackedView, sx, sy)
	}
	return lipgloss.JoinVertical(0, topView, centerView, footer)
}

func New(c context.AppContext) tea.Model {
	defaultRevSet, _ := c.RunCommandImmediate(jj.ConfigGet("revsets.log"))
	revisionsModel := revisions.New(c)
	previewModel := preview.New(c)
	statusModel := status.New(c)
	return Model{
		context:      c,
		keyMap:       c.KeyMap(),
		state:        common.Loading,
		revisions:    &revisionsModel,
		previewModel: &previewModel,
		status:       &statusModel,
		revsetModel:  revset.New(string(defaultRevSet)),
	}
}
