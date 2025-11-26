package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"

	"github.com/idursun/jjui/internal/ui/flash"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
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
	"github.com/idursun/jjui/internal/ui/redo"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/internal/ui/status"
	"github.com/idursun/jjui/internal/ui/undo"
)

var _ common.Model = (*Model)(nil)

type Model struct {
	*common.Sizeable
	revisions    *revisions.Model
	oplog        *oplog.Model
	revsetModel  *revset.Model
	previewModel *preview.Model
	diff         *diff.Model
	leader       *leader.Model
	flash        *flash.Model
	state        common.State
	status       *status.Model
	context      *context.MainContext
	keyMap       config.KeyMappings[key.Binding]
	stacked      common.Model
}

type triggerAutoRefreshMsg struct{}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(tea.SetWindowTitle(fmt.Sprintf("jjui - %s", m.context.Location)), m.revisions.Init(), m.scheduleAutoRefresh())
}

func (m *Model) handleFocusInputMessage(msg tea.Msg) (tea.Cmd, bool) {
	if _, ok := msg.(common.CloseViewMsg); ok {
		if m.leader != nil {
			m.leader = nil
			return nil, true
		}
		if m.diff != nil {
			m.diff = nil
			return nil, true
		}
		if m.stacked != nil {
			m.stacked = nil
			return nil, true
		}
		if m.oplog != nil {
			m.oplog = nil
			return common.SelectionChanged, true
		}
		return nil, false
	}

	if m.leader != nil {
		return m.leader.Update(msg), true
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.diff != nil {
			return m.diff.Update(msg), true
		}

		if m.revsetModel.Editing {
			m.state = common.Loading
			return m.revsetModel.Update(msg), true
		}

		if m.status.IsFocused() {
			return m.status.Update(msg), true
		}

		if m.revisions.IsEditing() {
			return m.revisions.Update(msg), true
		}

		if m.stacked != nil {
			return m.stacked.Update(msg), true
		}
	}

	return nil, false
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if cmd, handled := m.handleFocusInputMessage(msg); handled {
		return cmd
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.FocusMsg:
		return common.RefreshAndKeepSelections
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Cancel) && m.state == common.Error:
			m.state = common.Ready
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Cancel) && m.stacked != nil:
			m.stacked = nil
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Cancel) && m.flash.Any():
			m.flash.DeleteOldest()
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Quit) && m.isSafeToQuit():
			return tea.Quit
		case key.Matches(msg, m.keyMap.OpLog.Mode):
			m.oplog = oplog.New(m.context, m.Width, m.Height)
			return m.oplog.Init()
		case key.Matches(msg, m.keyMap.Revset) && m.revisions.InNormalMode():
			return m.revsetModel.Update(revset.EditRevSetMsg{Clear: m.state != common.Error})
		case key.Matches(msg, m.keyMap.Git.Mode) && m.revisions.InNormalMode():
			m.stacked = git.NewModel(m.context, m.revisions.SelectedRevisions(), m.Width, m.Height)
			return m.stacked.Init()
		case key.Matches(msg, m.keyMap.Undo) && m.revisions.InNormalMode():
			m.stacked = undo.NewModel(m.context)
			cmds = append(cmds, m.stacked.Init())
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Redo) && m.revisions.InNormalMode():
			m.stacked = redo.NewModel(m.context)
			cmds = append(cmds, m.stacked.Init())
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Bookmark.Mode) && m.revisions.InNormalMode():
			changeIds := m.revisions.GetCommitIds()
			m.stacked = bookmarks.NewModel(m.context, m.revisions.SelectedRevision(), changeIds, m.Width, m.Height)
			cmds = append(cmds, m.stacked.Init())
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Help):
			cmds = append(cmds, common.ToggleHelp)
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Mode, m.keyMap.Preview.ToggleBottom):
			if key.Matches(msg, m.keyMap.Preview.ToggleBottom) {
				previewPos := m.previewModel.AtBottom()
				m.previewModel.SetPosition(false, !previewPos)
				if m.previewModel.Visible() {
					return tea.Batch(cmds...)
				}
			}
			m.previewModel.ToggleVisible()
			cmds = append(cmds, common.SelectionChanged)
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Expand) && m.previewModel.Visible():
			m.previewModel.Expand()
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Shrink) && m.previewModel.Visible():
			m.previewModel.Shrink()
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.CustomCommands):
			m.stacked = customcommands.NewModel(m.context, m.Width, m.Height)
			cmds = append(cmds, m.stacked.Init())
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Leader):
			m.leader = leader.New(m.context)
			cmds = append(cmds, leader.InitCmd)
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.FileSearch.Toggle):
			rev := m.revisions.SelectedRevision()
			if rev == nil {
				// noop if current revset does not exist (#264)
				return nil
			}
			out, _ := m.context.RunCommandImmediate(jj.FilesInRevision(rev))
			return common.FileSearch(m.context.CurrentRevset, m.previewModel.Visible(), rev, out)
		case key.Matches(msg, m.keyMap.QuickSearch) && m.oplog != nil:
			// HACK: prevents quick search from activating in op log view
			return nil
		case key.Matches(msg, m.keyMap.Suspend):
			return tea.Suspend
		default:
			for _, command := range m.context.CustomCommands {
				if !command.IsApplicableTo(m.context.SelectedItem) {
					continue
				}
				if key.Matches(msg, command.Binding()) {
					return command.Prepare(m.context)
				}
			}
		}
	case common.ExecMsg:
		return exec_process.ExecLine(m.context, msg)
	case common.ToggleHelpMsg:
		if m.stacked == nil {
			m.stacked = helppage.New(m.context)
			if p, ok := m.stacked.(common.ISizeable); ok {
				p.SetHeight(m.Height - 2)
				p.SetWidth(m.Width)
			}
		} else {
			m.stacked = nil
		}
		return nil
	case common.ShowDiffMsg:
		m.diff = diff.New(string(msg), m.Width, m.Height)
		return m.diff.Init()
	case common.UpdateRevisionsSuccessMsg:
		m.state = common.Ready
	case triggerAutoRefreshMsg:
		return tea.Batch(m.scheduleAutoRefresh(), func() tea.Msg {
			return common.AutoRefreshMsg{}
		})
	case common.UpdateRevSetMsg:
		m.context.CurrentRevset = string(msg)
		m.revsetModel.AddToHistory(m.context.CurrentRevset)
		return common.Refresh
	case common.ShowPreview:
		m.previewModel.SetVisible(bool(msg))
		cmds = append(cmds, common.SelectionChanged)
		return tea.Batch(cmds...)
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if s, ok := m.stacked.(common.ISizeable); ok {
			s.SetWidth(m.Width - 2)
			s.SetHeight(m.Height - 2)
		}
		m.status.SetWidth(m.Width)
		m.revisions.SetHeight(m.Height)
		m.revisions.SetWidth(m.Width)
		m.revsetModel.SetWidth(m.Width)
		m.revsetModel.SetHeight(1)
	}

	if m.revsetModel.Editing {
		cmds = append(cmds, m.revsetModel.Update(msg))
	}

	cmds = append(cmds, m.status.Update(msg))
	cmds = append(cmds, m.flash.Update(msg))

	if m.stacked != nil {
		cmds = append(cmds, m.stacked.Update(msg))
	}

	if m.oplog != nil {
		cmds = append(cmds, m.oplog.Update(msg))
	} else {
		cmds = append(cmds, m.revisions.Update(msg))
	}

	if m.previewModel.Visible() {
		cmds = append(cmds, m.previewModel.Update(msg))
	}

	return tea.Batch(cmds...)
}

func (m *Model) updateStatus() {
	switch {
	case m.diff != nil:
		m.status.SetMode("diff")
		m.status.SetHelp(m.diff)
	case m.oplog != nil:
		m.status.SetMode("oplog")
		m.status.SetHelp(m.oplog)
	case m.stacked != nil:
		if s, ok := m.stacked.(help.KeyMap); ok {
			m.status.SetHelp(s)
		}
	case m.leader != nil:
		m.status.SetMode("leader")
		m.status.SetHelp(m.leader)
	default:
		m.status.SetHelp(m.revisions)
		m.status.SetMode(m.revisions.CurrentOperation().Name())
	}
}

func (m *Model) UpdatePreviewPosition() {
	if m.previewModel.AutoPosition() {
		atBottom := m.Height >= m.Width/2
		m.previewModel.SetPosition(true, atBottom)
	}
}

func (m *Model) View() string {
	m.updateStatus()
	footer := m.status.View()
	footerHeight := lipgloss.Height(footer)

	if m.diff != nil {
		m.diff.SetHeight(m.Height - footerHeight)
		return lipgloss.JoinVertical(0, m.diff.View(), footer)
	}

	topView := m.revsetModel.View()
	topViewHeight := lipgloss.Height(topView)

	m.UpdatePreviewPosition()

	bottomPreviewHeight := 0
	if m.previewModel.Visible() && m.previewModel.AtBottom() {
		bottomPreviewHeight = int(float64(m.Height) * (m.previewModel.WindowPercentage() / 100.0))
	}
	leftView := m.renderLeftView(footerHeight, topViewHeight, bottomPreviewHeight)
	centerView := leftView

	if m.previewModel.Visible() {
		if m.previewModel.AtBottom() {
			m.previewModel.SetWidth(m.Width)
			m.previewModel.SetHeight(bottomPreviewHeight)
		} else {
			m.previewModel.SetWidth(m.Width - lipgloss.Width(leftView))
			m.previewModel.SetHeight(m.Height - footerHeight - topViewHeight)
		}
		previewView := m.previewModel.View()
		if m.previewModel.AtBottom() {
			centerView = lipgloss.JoinVertical(lipgloss.Top, leftView, previewView)
		} else {
			centerView = lipgloss.JoinHorizontal(lipgloss.Left, leftView, previewView)
		}
	}

	if m.stacked != nil {
		stackedView := m.stacked.View()
		w, h := lipgloss.Size(stackedView)
		sx := (m.Width - w) / 2
		sy := (m.Height - h) / 2
		centerView = screen.Stacked(centerView, stackedView, sx, sy)
	}

	full := lipgloss.JoinVertical(0, topView, centerView, footer)
	flashMessageView := m.flash.View()
	if flashMessageView != "" {
		mw, mh := lipgloss.Size(flashMessageView)
		full = screen.Stacked(full, flashMessageView, m.Width-mw, m.Height-mh-1)
	}
	statusFuzzyView := m.status.FuzzyView()
	if statusFuzzyView != "" {
		_, mh := lipgloss.Size(statusFuzzyView)
		full = screen.Stacked(full, statusFuzzyView, 0, m.Height-mh-1)
	}
	return full
}

func (m *Model) renderLeftView(footerHeight int, topViewHeight int, bottomPreviewHeight int) string {
	leftView := ""
	w := m.Width
	h := 0

	if m.previewModel.Visible() {
		if m.previewModel.AtBottom() {
			h = bottomPreviewHeight
		} else {
			w = m.Width - int(float64(m.Width)*(m.previewModel.WindowPercentage()/100.0))
		}
	}

	if m.oplog != nil {
		m.oplog.SetWidth(w)
		m.oplog.SetHeight(m.Height - footerHeight - topViewHeight - h)
		leftView = m.oplog.View()
	} else {
		m.revisions.SetWidth(w)
		m.revisions.SetHeight(m.Height - footerHeight - topViewHeight - h)
		leftView = m.revisions.View()
	}
	return leftView
}

func (m *Model) scheduleAutoRefresh() tea.Cmd {
	interval := config.Current.UI.AutoRefreshInterval
	if interval > 0 {
		return tea.Tick(time.Duration(interval)*time.Second, func(time.Time) tea.Msg {
			return triggerAutoRefreshMsg{}
		})
	}
	return nil
}

func (m *Model) isSafeToQuit() bool {
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

var _ tea.Model = (*wrapper)(nil)

type wrapper struct {
	ui *Model
}

func (w wrapper) Init() tea.Cmd {
	return w.ui.Init()
}

func (w wrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return w, w.ui.Update(msg)
}

func (w wrapper) View() string {
	return w.ui.View()
}

func NewUI(c *context.MainContext) *Model {
	revisionsModel := revisions.New(c)
	previewModel := preview.New(c)
	statusModel := status.New(c)
	return &Model{
		Sizeable:     &common.Sizeable{Width: 0, Height: 0},
		context:      c,
		keyMap:       config.Current.GetKeyMap(),
		state:        common.Loading,
		revisions:    revisionsModel,
		previewModel: &previewModel,
		status:       &statusModel,
		revsetModel:  revset.New(c),
		flash:        flash.New(c),
	}
}

func New(c *context.MainContext) tea.Model {
	return wrapper{ui: NewUI(c)}
}
