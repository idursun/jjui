package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/helppage"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/revset"

	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/status"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	revisions      tea.Model
	revsetModel    revset.Model
	previewModel   tea.Model
	previewVisible bool
	helpPage       tea.Model
	diff           tea.Model
	state          common.State
	error          error
	status         tea.Model
	output         string
	width          int
	height         int
	context        context.AppContext
	keyMap         common.KeyMappings[key.Binding]
}

func (m Model) Init() tea.Cmd {
	return tea.Sequence(tea.SetWindowTitle("jjui"), m.revisions.Init())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(common.CloseViewMsg); ok && m.diff != nil {
		m.diff = nil
		return m, nil
	}

	var cmd tea.Cmd
	if m.diff != nil {
		m.diff, cmd = m.diff.Update(msg)
		return m, cmd
	}

	if m.revsetModel.Editing {
		m.revsetModel, cmd = m.revsetModel.Update(msg)
		return m, cmd
	}

	var cmds []tea.Cmd
	m.status, cmd = m.status.Update(msg)
	cmds = append(cmds, cmd)

	if r, ok := m.revisions.(common.Focusable); ok && r.IsFocused() {
		m.revisions, cmd = m.revisions.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	if r, ok := m.previewModel.(common.Focusable); ok && r.IsFocused() {
		m.previewModel, cmd = m.previewModel.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	revisionsModel, ok := m.revisions.(*revisions.Model)
	inOverlayEditing := ok && revisionsModel.InOverlayEditing()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !inOverlayEditing {
			switch {
			case key.Matches(msg, m.keyMap.Cancel) && m.state == common.Error:
				m.state = common.Ready
				m.error = nil
			case key.Matches(msg, m.keyMap.Cancel) && m.helpPage != nil:
				m.helpPage = nil
				m.error = nil
			case key.Matches(msg, m.keyMap.Revset):
				m.revsetModel, _ = m.revsetModel.Update(revset.EditRevSetMsg{})
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
		}
	case common.ToggleHelpMsg:
		if m.helpPage == nil {
			m.helpPage = helppage.New(m.context)
			if p, ok := m.helpPage.(common.Sizable); ok {
				p.SetHeight(m.height - 2)
				p.SetWidth(m.width)
			}
		} else {
			m.helpPage = nil
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
		m.error = msg
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if r, ok := m.revisions.(common.Sizable); ok {
			if m.previewVisible {
				r.SetWidth(m.width / 2)
			} else {
				r.SetWidth(m.width)
			}
			r.SetHeight(m.height - 4)
		}
		if p, ok := m.previewModel.(common.Sizable); ok && m.previewVisible {
			p.SetWidth(m.width / 2)
			p.SetHeight(m.height - 4)
		}
		if s, ok := m.status.(common.Sizable); ok {
			s.SetWidth(m.width)
		}
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
		topView += fmt.Sprintf("\nerror: %v\n", m.error)
	}
	topViewHeight := lipgloss.Height(topView)

	footer := m.status.View()
	footerHeight := lipgloss.Height(footer)

	if m.helpPage != nil {
		return lipgloss.JoinVertical(0, topView, m.helpPage.View(), footer)
	}

	if r, ok := m.revisions.(common.Sizable); ok {
		r.SetWidth(m.width)
		if m.previewVisible {
			r.SetWidth(m.width / 2)
			if p, ok := m.previewModel.(common.Focusable); ok && p.IsFocused() {
				r.SetWidth(4)
			}
		}
		r.SetHeight(m.height - footerHeight - topViewHeight)
	}
	revisionsView := m.revisions.View()

	previewView := ""
	if p, ok := m.previewModel.(common.Sizable); ok && m.previewVisible {
		p.SetWidth(m.width - lipgloss.Width(revisionsView))
		p.SetHeight(m.height - footerHeight - topViewHeight)
		previewView = m.previewModel.View()
	}

	centerView := lipgloss.JoinHorizontal(lipgloss.Left, revisionsView, previewView)
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
