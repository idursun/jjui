package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/bookmarks"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	customcommands "github.com/idursun/jjui/internal/ui/custom_commands"
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/exec_process"
	"github.com/idursun/jjui/internal/ui/git"
	"github.com/idursun/jjui/internal/ui/helppage"
	"github.com/idursun/jjui/internal/ui/leader"
	"github.com/idursun/jjui/internal/ui/oplog"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/internal/ui/status"
	"github.com/idursun/jjui/internal/ui/undo"
)

type Model struct {
	revisions               *revisions.Model
	oplog                   *oplog.Model
	revsetModel             *revset.Model
	previewModel            *preview.Model
	previewVisible          bool
	previewWindowPercentage float64
	diff                    *diff.Model
	leader                  *leader.Model
	state                   common.State
	error                   error
	status                  *status.Model
	output                  string
	width                   int
	height                  int
	context                 *context.MainContext
	keyMap                  config.KeyMappings[key.Binding]
	stacked                 tea.Model
}

type triggerAutoRefreshMsg struct{}

func (m Model) Init() tea.Cmd {
	return tea.Sequence(tea.SetWindowTitle(fmt.Sprintf("jjui - %s", m.context.Location)), m.revisions.Init(), m.scheduleAutoRefresh())
}

func (m Model) handleFocusInputMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	if x, ok := msg.(leader.PendingMsg); ok {
		return m, leader.TakePending(x), true
	}

	var cmd tea.Cmd
	if _, ok := msg.(common.CloseViewMsg); ok {
		if m.leader != nil {
			m.leader = nil
			return m, nil, true
		}
		if m.diff != nil {
			m.diff = nil
			return m, nil, true
		}
		if m.stacked != nil {
			m.stacked = nil
			return m, nil, true
		}
		if m.oplog != nil {
			m.oplog = nil
			return m, common.SelectionChanged, true
		}
		return m, nil, false
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.leader != nil {
			m.leader, cmd = m.leader.Update(msg)
			return m, cmd, true
		}

		if m.diff != nil {
			m.diff, cmd = m.diff.Update(msg)
			return m, cmd, true
		}

		if m.revsetModel.Editing {
			m.revsetModel, cmd = m.revsetModel.Update(msg)
			m.state = common.Loading
			return m, cmd, true
		}

		if m.status.IsFocused() {
			m.status, cmd = m.status.Update(msg)
			return m, cmd, true
		}

		if m.revisions.IsFocused() {
			m.revisions, cmd = m.revisions.Update(msg)
			return m, cmd, true
		}

		if m.stacked != nil {
			m.stacked, cmd = m.stacked.Update(msg)
			return m, cmd, true
		}
	}

	return m, nil, false
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m, cmd, handled := m.handleFocusInputMessage(msg); handled {
		return m, cmd
	}

	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Cancel) && m.state == common.Error:
			m.state = common.Ready
			m.error = nil
		case key.Matches(msg, m.keyMap.Cancel) && m.stacked != nil:
			m.stacked = nil
		case key.Matches(msg, m.keyMap.Quit) && m.isSafeToQuit():
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.OpLog.Mode):
			m.oplog = oplog.New(m.context, m.width, m.height)
			return m, m.oplog.Init()
		case key.Matches(msg, m.keyMap.Revset) && m.revisions.InNormalMode():
			m.revsetModel, _ = m.revsetModel.Update(revset.EditRevSetMsg{Clear: m.state != common.Error})
			return m, nil
		case key.Matches(msg, m.keyMap.Git.Mode) && m.revisions.InNormalMode():
			m.stacked = git.NewModel(m.context, m.revisions.SelectedRevision(), 80, 30)
			return m, m.stacked.Init()
		case key.Matches(msg, m.keyMap.Undo) && m.revisions.InNormalMode():
			m.stacked = undo.NewModel(m.context)
			cmds = append(cmds, m.stacked.Init())
		case key.Matches(msg, m.keyMap.Bookmark.Mode) && m.revisions.InNormalMode():
			changeIds := m.revisions.GetCommitIds()
			m.stacked = bookmarks.NewModel(m.context, m.revisions.SelectedRevision(), changeIds, 80, 30)
			cmds = append(cmds, m.stacked.Init())
		case key.Matches(msg, m.keyMap.Help):
			cmds = append(cmds, common.ToggleHelp)
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Mode):
			m.previewVisible = !m.previewVisible
			cmds = append(cmds, common.SelectionChanged)
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Expand):
			m.previewWindowPercentage += config.Current.Preview.WidthIncrementPercentage
		case key.Matches(msg, m.keyMap.Preview.Shrink):
			m.previewWindowPercentage -= config.Current.Preview.WidthIncrementPercentage
		case key.Matches(msg, m.keyMap.CustomCommands):
			m.stacked = customcommands.NewModel(m.context, 80, 30)
			cmds = append(cmds, m.stacked.Init())
		case key.Matches(msg, m.keyMap.Leader):
			m.leader = leader.New(m.context.Leader)
			cmds = append(cmds, m.leader.Init())
		case key.Matches(msg, m.keyMap.QuickSearch) && m.oplog != nil:
			// HACK: prevents quick search from activating in op log view
			return m, nil
		case key.Matches(msg, m.keyMap.Suspend):
			return m, tea.Suspend
		default:
			for _, command := range m.context.CustomCommands {
				if !command.IsApplicableTo(m.context.SelectedItem) {
					continue
				}
				if key.Matches(msg, command.Binding()) {
					return m, command.Prepare(m.context)
				}
			}
		}
	case common.ExecMsg:
		return m, exec_process.ExecLine(m.context, msg)
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
		return m, m.diff.Init()
	case common.CommandCompletedMsg:
		m.output = msg.Output
	case common.UpdateRevisionsFailedMsg:
		m.state = common.Error
		m.output = msg.Output
		m.error = msg.Err
	case common.UpdateRevisionsSuccessMsg:
		m.state = common.Ready
	case triggerAutoRefreshMsg:
		return m, tea.Batch(m.scheduleAutoRefresh(), func() tea.Msg {
			return common.AutoRefreshMsg{}
		})
	case common.UpdateRevSetMsg:
		var revsetCmd tea.Cmd
		m.context.CurrentRevset = string(msg)
		m.revsetModel, revsetCmd = m.revsetModel.Update(msg)
		var revisionsCmd tea.Cmd
		m.revisions, revisionsCmd = m.revisions.Update(msg)
		return m, tea.Batch(revsetCmd, revisionsCmd)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if s, ok := m.stacked.(common.Sizable); ok {
			s.SetWidth(m.width - 2)
			s.SetHeight(m.height - 2)
		}
		m.status.SetWidth(m.width)
		m.revisions.SetHeight(m.height)
		m.revisions.SetWidth(m.width)
	}

	if m.revsetModel.Editing {
		m.revsetModel, cmd = m.revsetModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.status, cmd = m.status.Update(msg)
	cmds = append(cmds, cmd)

	if m.stacked != nil {
		m.stacked, cmd = m.stacked.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.oplog != nil {
		m.oplog, cmd = m.oplog.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.revisions, cmd = m.revisions.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.previewVisible {
		m.previewModel, cmd = m.previewModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.diff != nil {
		m.status.SetMode("diff")
		m.status.SetHelp(m.diff)
		footer := m.status.View()
		footerHeight := lipgloss.Height(footer)
		m.diff.SetHeight(m.height - footerHeight)
		return lipgloss.JoinVertical(0, m.diff.View(), footer)
	}

	topView := m.revsetModel.View()
	if m.state == common.Error {
		topView += fmt.Sprintf("\n%s\n", m.output)
	}
	topViewHeight := lipgloss.Height(topView)

	if m.oplog != nil {
		m.status.SetMode("oplog")
		m.status.SetHelp(m.oplog)
	} else {
		m.status.SetHelp(m.revisions)
		m.status.SetMode(m.revisions.CurrentOperation().Name())
	}

	if m.leader != nil {
		m.status.SetMode("leader")
		m.status.SetHelp(m.leader)
	}

	footer := m.status.View()
	footerHeight := lipgloss.Height(footer)

	leftView := m.renderLeftView(footerHeight, topViewHeight)

	previewView := ""
	if m.previewVisible {
		m.previewModel.SetWidth(m.width - lipgloss.Width(leftView))
		m.previewModel.SetHeight(m.height - footerHeight - topViewHeight)
		previewView = m.previewModel.View()
	}

	centerView := lipgloss.JoinHorizontal(lipgloss.Left, leftView, previewView)

	if m.stacked != nil {
		stackedView := m.stacked.View()
		w, h := lipgloss.Size(stackedView)
		sx := (m.width - w) / 2
		sy := (m.height - h) / 2
		centerView = screen.Stacked(centerView, stackedView, sx, sy)
	}
	return lipgloss.JoinVertical(0, topView, centerView, footer)
}

func (m Model) renderLeftView(footerHeight int, topViewHeight int) string {
	leftView := ""
	w := m.width

	if m.previewVisible {
		w = m.width - int(float64(m.width)*(m.previewWindowPercentage/100.0))
	}

	if m.oplog != nil {
		m.oplog.SetWidth(w)
		m.oplog.SetHeight(m.height - footerHeight - topViewHeight)
		leftView = m.oplog.View()
	} else {
		m.revisions.SetWidth(w)
		m.revisions.SetHeight(m.height - footerHeight - topViewHeight)
		leftView = m.revisions.View()
	}
	return leftView
}

func (m Model) scheduleAutoRefresh() tea.Cmd {
	interval := config.Current.UI.AutoRefreshInterval
	if interval > 0 {
		return tea.Tick(time.Duration(interval)*time.Second, func(time.Time) tea.Msg {
			return triggerAutoRefreshMsg{}
		})
	}
	return nil
}

func (m Model) isSafeToQuit() bool {
	if m.stacked != nil {
		return false
	}
	if m.oplog != nil {
		return false
	}
	if m.revisions.CurrentOperation().Name() == "normal" {
		return true
	}
	return false
}

func New(c *context.MainContext) tea.Model {
	revisionsModel := revisions.New(c)
	previewModel := preview.New(c)
	statusModel := status.New(c)
	return Model{
		context:                 c,
		keyMap:                  config.Current.GetKeyMap(),
		state:                   common.Loading,
		revisions:               &revisionsModel,
		previewModel:            &previewModel,
		previewVisible:          config.Current.Preview.ShowAtStart,
		previewWindowPercentage: config.Current.Preview.WidthPercentage,
		status:                  &statusModel,
		revsetModel:             revset.New(c),
	}
}
