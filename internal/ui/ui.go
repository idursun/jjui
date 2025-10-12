package ui

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/bookmarks"
	"github.com/idursun/jjui/internal/ui/git"
	"github.com/idursun/jjui/internal/ui/helppage"
	"github.com/idursun/jjui/internal/ui/undo"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/idursun/jjui/internal/ui/flash"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/exec_process"
	"github.com/idursun/jjui/internal/ui/oplog"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/internal/ui/status"
)

type Model struct {
	*common.Sizeable
	router    *view.Router
	revisions *revisions.Model
	flash     *flash.Model
	state     common.State
	status    *status.Model
	context   *context.MainContext
	actions   []*actions.ActionBinding
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.router.Init(), tea.SetWindowTitle(fmt.Sprintf("jjui - %s", m.context.Location)), m.revisions.Init(), m.registerAutoEvents())
}

func (m Model) registerAutoEvents() tea.Cmd {
	var cmds []tea.Cmd
	for k := range actions.Registry {
		if !strings.HasPrefix(k, "#") {
			continue
		}
		cmds = append(cmds, m.registerAutoEvent(k))
	}
	return tea.Batch(cmds...)
}

func (m Model) registerAutoEvent(autoEvent string) tea.Cmd {
	var cmds []tea.Cmd
	if action, ok := actions.Registry[autoEvent]; ok {
		events := action.GetArgs("on")
		for _, event := range events {
			cmds = append(cmds, m.router.AddWaiter(event, action))
		}
		if v, ok := action.Args["interval"]; ok {
			interval := v.(int64)
			if interval > 0 {
				cmds = append(cmds, tea.Tick(time.Duration(interval)*time.Second, func(time.Time) tea.Msg {
					log.Printf("Scheduling action %s on interval %d", action.Id, interval)
					return action.GetNextMsg()
				}))
			}

		}
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.context.Set(jj.RevsetPlaceholder, m.context.CurrentRevset)
	m.context.Set(jj.ChangeIdPlaceholder, m.router.Read(jj.ChangeIdPlaceholder))
	m.context.Set(jj.CommitIdPlaceholder, m.router.Read(jj.CommitIdPlaceholder))
	m.context.Set(jj.OperationIdPlaceholder, m.router.Read(jj.OperationIdPlaceholder))
	m.context.Set(jj.FilePlaceholder, m.router.Read(jj.FilePlaceholder))
	m.context.Set(jj.CheckedFilesPlaceholder, m.router.Read(jj.CheckedFilesPlaceholder))
	m.context.Set(jj.CheckedCommitIdsPlaceholder, m.router.Read(jj.CheckedCommitIdsPlaceholder))

	if msg, ok := msg.(actions.InvokeActionMsg); ok {
		// clearing the available actions since user has made a choice
		m.actions = nil
		if msg.Action.Id == "run" {
			args := msg.Action.GetArgs("jj")
			async := msg.Action.Get("async", false).(bool)
			if async {
				return m, m.context.RunCommand(jj.TemplatedArgs(args, m.context.GetVariables()))
			}

			interactive := msg.Action.Get("interactive", false).(bool)
			if interactive {
				cmd := m.context.RunInteractiveCommand(jj.TemplatedArgs(args, m.context.GetVariables()), common.Refresh)
				return m, tea.Sequence(cmd, msg.Action.GetNextCmd())
			}

			return m, func() tea.Msg {
				output, _ := m.context.RunCommandImmediate(jj.TemplatedArgs(args, m.context.GetVariables()))
				m.context.Set("$output", string(output))
				return msg.Action.GetNextMsg()
			}
		}
		if strings.HasPrefix(msg.Action.Id, "register ") {
			action := strings.TrimPrefix(msg.Action.Id, "register ")
			return m, m.registerAutoEvent(action)
		}
	}

	var cmd tea.Cmd
	var nm Model
	nm, cmd = m.internalUpdate(msg)

	var cmds []tea.Cmd
	cmds = append(cmds, cmd)

	nm.router, cmd = nm.router.Update(msg)
	cmds = append(cmds, cmd)

	//nm.status, cmd = nm.status.Update(msg)
	//cmds = append(cmds, cmd)

	nm.flash, cmd = nm.flash.Update(msg)
	cmds = append(cmds, cmd)

	if action, ok := msg.(actions.InvokeActionMsg); ok && strings.HasPrefix(action.Action.Id, "wait") == false {
		// scheduling the next action in the chain. This needs to be done outside the router so that the sub routers don't double schedule it
		cmds = append(cmds, action.Action.GetNextCmd())
	}

	return nm, tea.Batch(cmds...)
}

var cancelKey = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))

func (m Model) internalUpdate(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		if strings.HasPrefix(msg.Action.Id, "choose ") {
			scope := strings.TrimPrefix(msg.Action.Id, "choose ")
			var items []string
			if arg, ok := msg.Action.Args["items"]; ok {
				switch arg := arg.(type) {
				case string:
					items = strings.Split(m.context.ReplaceWithVariables(arg), "\n")
				case []string:
					items = arg
				case []interface{}:
					items = msg.Action.GetArgs("items")
				}
			}
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(view.Scope(scope), view.NewSimpleList(func(name, value string) {
				m.context.Set(name, value)
			}, scope, items))
			return m, cmd
		}

		switch msg.Action.Id {
		case "open oplog":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(view.ScopeOplog, oplog.New(m.context, m.Width, m.Height))
			return m, cmd
		case "open diff":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(view.ScopeDiff, diff.New(m.context, m.Width, m.Height))
			return m, cmd
		case "open undo":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(view.ScopeUndo, undo.NewModel(m.context))
			return m, cmd
		case "open bookmarks":
			changeIds := m.revisions.GetCommitIds()
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(view.ScopeBookmarks, bookmarks.NewModel(m.context, m.revisions.SelectedRevision(), changeIds, m.Width, m.Height))
			return m, cmd
		case "open git":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(view.ScopeGit, git.NewModel(m.context, m.revisions.SelectedRevisions(), m.Width, m.Height))
			return m, cmd
		case "open help":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(view.ScopeHelp, helppage.New(m.context))
			return m, cmd
		case "open exec_jj":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(view.ScopeExecJJ, exec_process.NewModel(m.context, view.ScopeExecJJ))
			return m, cmd
		case "open exec_sh":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(view.ScopeExecSh, exec_process.NewModel(m.context, view.ScopeExecSh))
			return m, cmd
		case "toggle preview":
			if m.router.Views[view.ScopePreview] != nil {
				delete(m.router.Views, view.ScopePreview)
				m.router.Scope = view.ScopeRevisions
				return m, nil
			}
			model := preview.New(m.context)
			m.router.Views[view.ScopePreview] = model
			return m, model.Init()
		case "refresh":
			return m, common.RefreshAndKeepSelections
		case "suspend":
			return m, tea.Suspend
		case "quit":
			return m, tea.Quit
		}
	case tea.FocusMsg:
		return m, common.RefreshAndKeepSelections
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, cancelKey) && m.state == common.Error:
			m.state = common.Ready
			return m, nil
		case key.Matches(msg, cancelKey) && m.flash.Any():
			m.flash.DeleteOldest()
			return m, nil
			//case key.Matches(msg, m.keyMap.FileSearch.Toggle):
			//	rev := m.revisions.SelectedRevision()
			//	if rev == nil {
			//		// noop if current revset does not exist (#264)
			//		return m, nil
			//	}
			//	out, _ := m.context.RunCommandImmediate(jj.FilesInRevision(rev))
			//	return m, common.FileSearch(m.context.CurrentRevset, false, rev, out)
		}
	case common.ExecMsg:
		return m, exec_process.ExecLine(m.context, msg)
	case common.UpdateRevisionsSuccessMsg:
		m.state = common.Ready
	case common.UpdateRevSetMsg:
		m.context.CurrentRevset = string(msg)
		//m.revsetModel.AddToHistory(m.context.CurrentRevset)
		return m, common.Refresh
	case common.ShowAvailableBindingMatches:
		m.actions = msg.Matches
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.status.SetWidth(m.Width)
		m.revisions.SetHeight(m.Height)
		m.revisions.SetWidth(m.Width)
	}
	return m, nil
}

func (m Model) updateStatus() {
	model := m.router.Views[m.router.Scope]
	if h, ok := model.(view.IStatus); ok {
		m.status.SetMode(h.Name())
		m.status.SetHelp(h)
	} else {
		m.status.SetHelp(m.revisions)
		m.status.SetMode(m.revisions.Name())
	}
}

func (m Model) View() string {
	m.updateStatus()
	var statusModel tea.Model
	if v, ok := m.router.Views[view.ScopeExecSh]; ok {
		statusModel = v
	} else if v, ok := m.router.Views[view.ScopeExecJJ]; ok {
		statusModel = v
	} else {
		statusModel = m.status
	}

	footer := statusModel.View()
	footerHeight := lipgloss.Height(footer)

	if diffView, ok := m.router.Views[view.ScopeDiff]; ok {
		if d, ok := diffView.(common.ISizeable); ok {
			d.SetWidth(m.Width)
			d.SetHeight(m.Height - footerHeight)
		}
		return lipgloss.JoinVertical(0, diffView.View(), footer)
	}

	topView := m.router.Views[view.ScopeRevset].View()
	topViewHeight := lipgloss.Height(topView)

	bottomPreviewHeight := 0
	leftView := m.renderLeftView(footerHeight, topViewHeight, bottomPreviewHeight)
	centerView := leftView
	previewModel := m.router.Views[view.ScopePreview]

	if previewModel != nil {
		if p, ok := previewModel.(common.ISizeable); ok {
			p.SetWidth(m.Width - lipgloss.Width(leftView))
			p.SetHeight(m.Height - footerHeight - topViewHeight)
		}
		previewView := previewModel.View()
		centerView = lipgloss.JoinHorizontal(lipgloss.Left, leftView, previewView)
	}

	var stacked tea.Model
	if strings.HasPrefix(string(m.router.Scope), "list ") {
		stacked = m.router.Views[m.router.Scope]
	} else if v, ok := m.router.Views[view.ScopeUndo]; ok {
		stacked = v
	} else if v, ok := m.router.Views[view.ScopeBookmarks]; ok {
		stacked = v
	} else if v, ok := m.router.Views[view.ScopeGit]; ok {
		stacked = v
	} else if v, ok := m.router.Views[view.ScopeHelp]; ok {
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
	if len(m.actions) > 0 {
		shortcutStyle := common.DefaultPalette.Get("shortcut")
		textStyle := common.DefaultPalette.Get("text")
		actionsView := actions.RenderAvailableActions(m.actions, textStyle, shortcutStyle)
		w, h := lipgloss.Size(actionsView)
		full = screen.Stacked(full, actionsView, (m.Width-w)/2, (m.Height-h)/2)
	}
	return full
}

func (m Model) renderLeftView(footerHeight int, topViewHeight int, bottomPreviewHeight int) string {
	w := m.Width
	if _, ok := m.router.Views[view.ScopePreview]; ok {
		w = m.Width - int(float64(m.Width)*(50/100.0))
	}

	var model tea.Model
	if oplog, ok := m.router.Views[view.ScopeOplog]; ok {
		model = oplog
	} else {
		model = m.router.Views[view.ScopeRevisions]
	}

	if s, ok := model.(common.ISizeable); ok {
		s.SetWidth(w)
		s.SetHeight(m.Height - footerHeight - topViewHeight - bottomPreviewHeight)
	}
	return model.View()
}

func New(c *context.MainContext) tea.Model {
	revisionsModel := revisions.New(c)
	statusModel := status.New(c)
	revsetModel := revset.New(c)
	c.Router.Views = map[view.Scope]tea.Model{
		view.ScopeRevisions: revisionsModel,
		view.ScopeRevset:    revsetModel,
		view.ScopeStatus:    statusModel,
	}
	m := Model{
		Sizeable:  &common.Sizeable{Width: 0, Height: 0},
		context:   c,
		state:     common.Loading,
		revisions: revisionsModel,
		status:    statusModel,
		flash:     flash.New(c),
		router:    c.Router,
	}
	return m
}
