package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/undo"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/idursun/jjui/internal/ui/flash"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/exec_process"
	"github.com/idursun/jjui/internal/ui/leader"
	"github.com/idursun/jjui/internal/ui/oplog"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/internal/ui/status"
)

type Model struct {
	*common.Sizeable
	router    view.Router
	revisions *revisions.Model
	leader    *leader.Model
	flash     *flash.Model
	state     common.State
	status    *status.Model
	context   *context.MainContext
	keyMap    config.KeyMappings[key.Binding]
	waiters   map[string]actions.WaitChannel
}

type triggerAutoRefreshMsg struct{}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.router.Init(), tea.SetWindowTitle(fmt.Sprintf("jjui - %s", m.context.Location)), m.revisions.Init(), m.scheduleAutoRefresh())
}

func (m Model) handleFocusInputMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	var cmd tea.Cmd
	if _, ok := msg.(common.CloseViewMsg); ok {
		if m.leader != nil {
			m.leader = nil
			return m, nil, true
		}
		return m, nil, false
	}

	if m.leader != nil {
		m.leader, cmd = m.leader.Update(msg)
		return m, cmd, true
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.status.IsFocused() {
			m.status, cmd = m.status.Update(msg)
			return m, cmd, true
		}
	}

	return m, nil, false
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var nm tea.Model
	nm, cmd = m.internalUpdate(msg)
	if msg, ok := msg.(actions.InvokeActionMsg); ok {
		if len(m.waiters) > 0 {
			for k, ch := range m.waiters {
				if k == msg.Action.Id {
					ch <- actions.WaitResultContinue
					close(ch)
					delete(m.waiters, k)
					return m, cmd
				}
			}
		}
		if strings.HasPrefix(msg.Action.Id, "wait") {
			message := strings.TrimPrefix(msg.Action.Id, "wait ")
			var waitCmd tea.Cmd
			m.waiters[message], waitCmd = msg.Action.Wait()
			return m, tea.Batch(cmd, waitCmd)
		}
		cmd = tea.Sequence(cmd, msg.Action.GetNext())
	}
	m.context.ScopeValues = map[string]string{}
	m.context.UpdateScopeValues(m.revisions.GetContext())
	return nm, cmd
}

func (m Model) internalUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		switch msg.Action.Id {
		case "ui.oplog":
			oplog := oplog.New(m.context, m.Width, m.Height)
			m.router.Scope = actions.ScopeOplog
			m.router.Views[actions.ScopeOplog] = oplog
			return m, oplog.Init()
		case "ui.diff":
			m.router.Scope = actions.ScopeDiff
			m.router.Views[actions.ScopeDiff] = diff.New("", m.Width, m.Height)
			return m, m.router.Views[m.router.Scope].Init()
		case "ui.undo":
			m.router.Views[actions.ScopeUndo] = undo.NewModel(m.context)
			m.router.Scope = actions.ScopeUndo
			return m, m.router.Views[m.router.Scope].Init()
		case "ui.toggle_preview":
			if m.router.Views[actions.ScopePreview] != nil {
				delete(m.router.Views, actions.ScopePreview)
				m.router.Scope = actions.ScopeRevisions
				return m, nil
			}
			model := preview.New(m.context)
			m.router.Views[actions.ScopePreview] = model
			return m, model.Init()
		case "ui.refresh":
			return m, common.RefreshAndKeepSelections
		case "ui.quit":
			if m.isSafeToQuit() {
				return m, tea.Quit
			}
			return m, nil
		}
	case tea.FocusMsg:
		return m, common.RefreshAndKeepSelections
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Cancel) && m.state == common.Error:
			m.state = common.Ready
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Cancel) && m.flash.Any():
			m.flash.DeleteOldest()
			return m, tea.Batch(cmds...)
		//case key.Matches(msg, m.keyMap.Git.Mode) && m.revisions.InNormalMode():
		//	m.stacked = git.NewModel(m.context, m.revisions.SelectedRevision(), m.Width, m.Height)
		//	return m, m.stacked.Init()
		//case key.Matches(msg, m.keyMap.Undo) && m.revisions.InNormalMode():
		//	m.stacked = undo.NewModel(m.context)
		//	cmds = append(cmds, m.stacked.Init())
		//	return m, tea.Batch(cmds...)
		//case key.Matches(msg, m.keyMap.Bookmark.Mode):
		//	changeIds := m.revisions.GetCommitIds()
		//	m.router.Scope = common.ScopeBookmarks
		//	m.router.Views[common.ScopeBookmarks] = bookmarks.NewModel(m.context, m.revisions.SelectedRevision(), changeIds, m.Width, m.Height)
		//	return m, m.router.Views[m.router.Scope].Init()
		case key.Matches(msg, m.keyMap.Help):
			cmds = append(cmds, common.ToggleHelp)
			return m, tea.Batch(cmds...)
		//case key.Matches(msg, m.keyMap.Preview.Mode, m.keyMap.Preview.ToggleBottom):
		//	if key.Matches(msg, m.keyMap.Preview.ToggleBottom) {
		//		m.previewModel.TogglePosition()
		//		if m.previewModel.Visible() {
		//			return m, tea.Batch(cmds...)
		//		}
		//	}
		//	m.previewModel.ToggleVisible()
		//	cmds = append(cmds, common.SelectionChanged)
		//	return m, tea.Batch(cmds...)
		//case key.Matches(msg, m.keyMap.Preview.Expand) && m.previewModel.Visible():
		//	m.previewModel.Expand()
		//	return m, tea.Batch(cmds...)
		//case key.Matches(msg, m.keyMap.Preview.Shrink) && m.previewModel.Visible():
		//	m.previewModel.Shrink()
		//	return m, tea.Batch(cmds...)
		//case key.Matches(msg, m.keyMap.CustomCommands):
		//	m.stacked = customcommands.NewModel(m.context, m.Width, m.Height)
		//	cmds = append(cmds, m.stacked.Init())
		//	return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Leader):
			m.leader = leader.New(m.context)
			cmds = append(cmds, leader.InitCmd)
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.FileSearch.Toggle):
			rev := m.revisions.SelectedRevision()
			if rev == nil {
				// noop if current revset does not exist (#264)
				return m, nil
			}
			out, _ := m.context.RunCommandImmediate(jj.FilesInRevision(rev))
			return m, common.FileSearch(m.context.CurrentRevset, false, rev, out)
		case key.Matches(msg, m.keyMap.Suspend):
			return m, tea.Suspend
		default:
			m.router, cmd = m.router.Update(msg)
			return m, cmd
		}
	case common.ExecMsg:
		return m, exec_process.ExecLine(m.context, msg)
	case common.UpdateRevisionsSuccessMsg:
		m.state = common.Ready
	case triggerAutoRefreshMsg:
		return m, tea.Batch(m.scheduleAutoRefresh(), func() tea.Msg {
			return common.AutoRefreshMsg{}
		})
	case common.UpdateRevSetMsg:
		m.context.CurrentRevset = string(msg)
		//m.revsetModel.AddToHistory(m.context.CurrentRevset)
		return m, common.Refresh
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		//if s, ok := m.stacked.(common.ISizeable); ok {
		//	s.SetWidth(m.Width - 2)
		//	s.SetHeight(m.Height - 2)
		//}
		m.status.SetWidth(m.Width)
		m.revisions.SetHeight(m.Height)
		m.revisions.SetWidth(m.Width)
		//m.revsetModel.SetWidth(m.Width)
		//m.revsetModel.SetHeight(1)
	}

	m.router, cmd = m.router.Update(msg)
	cmds = append(cmds, cmd)

	m.status, cmd = m.status.Update(msg)
	cmds = append(cmds, cmd)

	m.flash, cmd = m.flash.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) updateStatus() {
	switch {
	case m.leader != nil:
		m.status.SetMode("leader")
		m.status.SetHelp(m.leader)
	default:
		model := m.router.Views[m.router.Scope]
		if h, ok := model.(help.KeyMap); ok {
			m.status.SetMode(string(m.router.Scope))
			m.status.SetHelp(h)
		} else {
			m.status.SetHelp(m.revisions)
			m.status.SetMode(m.revisions.CurrentOperation().Name())
		}
	}
}

func (m Model) View() string {
	m.updateStatus()
	footer := m.status.View()
	footerHeight := lipgloss.Height(footer)

	if diffView, ok := m.router.Views[actions.ScopeDiff]; ok {
		if d, ok := diffView.(common.ISizeable); ok {
			d.SetWidth(m.Width)
			d.SetHeight(m.Height - footerHeight)
		}
		return lipgloss.JoinVertical(0, diffView.View(), footer)
	}

	topView := m.router.Views[actions.ScopeRevset].View()
	topViewHeight := lipgloss.Height(topView)

	bottomPreviewHeight := 0
	//if m.previewModel.Visible() && m.previewModel.AtBottom() {
	//	bottomPreviewHeight = int(float64(m.Height) * (m.previewModel.WindowPercentage() / 100.0))
	//}
	leftView := m.renderLeftView(footerHeight, topViewHeight, bottomPreviewHeight)
	centerView := leftView
	previewModel := m.router.Views[actions.ScopePreview]

	if previewModel != nil {
		if p, ok := previewModel.(common.ISizeable); ok {
			p.SetWidth(m.Width - lipgloss.Width(leftView))
			p.SetHeight(m.Height - footerHeight - topViewHeight)
		}
		//if m.previewModel.AtBottom() {
		//	m.previewModel.SetWidth(m.Width)
		//	m.previewModel.SetHeight(bottomPreviewHeight)
		//} else {
		//	m.previewModel.SetWidth(m.Width - lipgloss.Width(leftView))
		//	m.previewModel.SetHeight(m.Height - footerHeight - topViewHeight)
		//}
		previewView := previewModel.View()
		//if m.previewModel.AtBottom() {
		//	centerView = lipgloss.JoinVertical(lipgloss.Top, leftView, previewView)
		//} else {
		centerView = lipgloss.JoinHorizontal(lipgloss.Left, leftView, previewView)
		//}
	}

	var stacked tea.Model
	if v, ok := m.router.Views[actions.ScopeUndo]; ok {
		stacked = v
	}

	if stacked != nil {
		stackedView := stacked.View()
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

func (m Model) renderLeftView(footerHeight int, topViewHeight int, bottomPreviewHeight int) string {
	w := m.Width
	//h := 0

	if _, ok := m.router.Views[actions.ScopePreview]; ok {
		w = m.Width - int(float64(m.Width)*(50/100.0))
	}
	//if m.previewModel.Visible() {
	//	if m.previewModel.AtBottom() {
	//		h = bottomPreviewHeight
	//	} else {
	//		w = m.Width - int(float64(m.Width)*(m.previewModel.WindowPercentage()/100.0))
	//	}
	//}

	var model tea.Model

	if oplog, ok := m.router.Views[actions.ScopeOplog]; ok {
		model = oplog
	} else {
		model = m.router.Views[actions.ScopeRevisions]
	}

	if s, ok := model.(common.ISizeable); ok {
		s.SetWidth(w)
		s.SetHeight(m.Height - footerHeight - topViewHeight - bottomPreviewHeight)
	}
	return model.View()
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
	if m.revisions.CurrentOperation().Name() == "normal" {
		return true
	}
	return false
}

func New(c *context.MainContext) tea.Model {
	revisionsModel := revisions.New(c)
	statusModel := status.New(c)
	revsetModel := revset.New(c)
	router := view.NewRouter(actions.ScopeRevisions)
	router.Views = map[actions.Scope]tea.Model{
		actions.ScopeRevisions: revisionsModel,
		actions.ScopeRevset:    revsetModel,
	}
	m := Model{
		Sizeable:  &common.Sizeable{Width: 0, Height: 0},
		context:   c,
		keyMap:    config.Current.GetKeyMap(),
		state:     common.Loading,
		revisions: revisionsModel,
		status:    &statusModel,
		flash:     flash.New(c),
		waiters:   make(map[string]actions.WaitChannel),
		router:    router,
	}
	c.ReadFn = func(value string) string {
		return m.router.Read(value)
	}
	return m
}
