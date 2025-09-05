package ui

import (
	"fmt"
	"time"

	"github.com/idursun/jjui/internal/ui/flash"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	customcommands "github.com/idursun/jjui/internal/ui/custom_commands"
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/leader"
	"github.com/idursun/jjui/internal/ui/oplog"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/internal/ui/status"
)

type Model struct {
	*view.BaseView
	diff                *diff.Model
	leader              *leader.Model
	flash               *flash.Model
	state               common.State
	context             *context.MainContext
	keyMap              config.KeyMappings[key.Binding]
	stacked             tea.Model
	stackedContainer    *view.StackedLayoutContainer
	horizontalContainer *view.LayoutContainer
}

type triggerAutoRefreshMsg struct{}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, tea.SetWindowTitle(fmt.Sprintf("jjui - %s", m.context.Location)))
	cmds = append(cmds, m.scheduleAutoRefresh())
	cmds = append(cmds, m.BaseView.Init())
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	if _, ok := msg.(common.CloseViewMsg); ok {
		if m.leader != nil {
			m.leader = nil
			m.context.ActiveList = view.ListRevisions
			return m, nil
		}
		if m.diff != nil {
			m.diff = nil
			m.context.ActiveList = view.ListRevisions
			return m, nil
		}
		if m.stacked != nil {
			m.stacked = nil
			m.context.ActiveList = view.ListRevisions
			return m, nil
		}
		m.BaseView.Model, cmd = m.BaseView.Update(msg)
		// Check if there are multiple views in the stacked container
		top := m.stackedContainer.Top()
		if top != nil && len(m.stackedContainer.GetRegistry()) > 1 {
			m.stackedContainer.Pop()
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Cancel) && m.state == common.Error:
			m.state = common.Ready
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Cancel) && m.stacked != nil:
			m.stacked = nil
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Cancel) && m.flash.Any():
			m.flash.DeleteOldest()
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Quit) && m.isSafeToQuit():
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.OpLog.Mode):
			v := oplog.New(m.context, m.Width, m.Height)
			v.Visible = true
			v.Focused = true
			m.stackedContainer.Push(v, view.Grow(1))
			return m, v.Init()
		case key.Matches(msg, m.keyMap.Help):
			cmds = append(cmds, common.ToggleHelp)
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Mode, m.keyMap.Preview.ToggleBottom):
			if key.Matches(msg, m.keyMap.Preview.ToggleBottom) {
				m.context.Preview.TogglePosition()
				if m.context.Preview.Visible {
					return m, tea.Batch(cmds...)
				}
			}
			m.context.Preview.ToggleVisible()
			if m.context.Preview.Visible {
				v := preview.New(m.context)
				v.Visible = true
				v.Focused = false
				m.horizontalContainer.Add(v, view.Grow(1))
				cmds = append(cmds, v.Init())
			} else {
				m.horizontalContainer.Remove("preview")
			}
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Expand) && m.context.Preview.Visible:
			m.context.Preview.Expand()
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Shrink) && m.context.Preview.Visible:
			m.context.Preview.Shrink()
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.CustomCommands):
			m.stacked = customcommands.NewModel(m.context, m.Width, m.Height)
			cmds = append(cmds, m.stacked.Init())
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Leader):
			m.leader = leader.New(m.context)
			cmds = append(cmds, leader.InitCmd)
			return m, tea.Batch(cmds...)
		//case key.Matches(msg, m.keyMap.QuickSearch) && m.oplog != nil:
		//	// HACK: prevents quick search from activating in op log view
		//	return m, nil
		case key.Matches(msg, m.keyMap.Suspend):
			return m, tea.Suspend
		default:
			m.BaseView.Model, cmd = m.BaseView.Update(msg)
			cmds = append(cmds, cmd)
		}

	default:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.Width = msg.Width
			m.Height = msg.Height
			if s, ok := m.stacked.(common.ISizeable); ok {
				s.SetWidth(m.Width - 2)
				s.SetHeight(m.Height - 2)
			}
			m.BaseView.SetWidth(msg.Width)
			m.BaseView.SetHeight(msg.Height)
		}
		m.BaseView.Model, cmd = m.BaseView.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)

	//var cmd tea.Cmd
	//var cmds []tea.Cmd
	//switch msg := msg.(type) {
	//case tea.KeyMsg:
	//	switch {
	//case key.Matches(msg, m.keyMap.Revset) && m.revisions.InNormalMode():
	//	m.revsetModel.Model, _ = m.revsetModel.Update(revset.EditRevSetMsg{Clear: m.state != common.Error})
	//	return m, nil
	//case key.Matches(msg, m.keyMap.Git.Mode) && m.revisions.InNormalMode():
	//	m.stacked = git.NewModel(m.context, m.revisions.SelectedRevision(), m.Width, m.Height)
	//	return m, m.stacked.Init()
	//case key.Matches(msg, m.keyMap.Undo) && m.revisions.InNormalMode():
	//	m.stacked = undo.NewModel(m.context)
	//	cmds = append(cmds, m.stacked.Init())
	//	return m, tea.Batch(cmds...)
	//case key.Matches(msg, m.keyMap.Bookmark.Mode) && m.revisions.InNormalMode():
	//	changeIds := m.revisions.GetCommitIds()
	//	m.stacked = bookmarks.NewModel(m.context, m.revisions.SelectedRevision(), changeIds, m.Width, m.Height)
	//	cmds = append(cmds, m.stacked.Init())
	//	return m, tea.Batch(cmds...)
	//case key.Matches(msg, m.keyMap.FileSearch.Toggle):
	//	rev := m.revisions.SelectedRevision()
	//	if rev == nil {
	//		// noop if current revset does not exist (#264)
	//		return m, nil
	//	}
	//	out, _ := m.context.RunCommandImmediate(jj.FilesInRevision(rev))
	//	return m, common.FileSearch(m.context.CurrentRevset, m.context.Preview.Visible, rev, out)
	//	default:
	//		for _, command := range m.context.CustomCommands {
	//			if !command.IsApplicableTo(m.context) {
	//				continue
	//			}
	//			if key.Matches(msg, command.Binding()) {
	//				return m, command.Prepare(m.context)
	//			}
	//		}
	//	}
	//case common.ExecMsg:
	//	return m, exec_process.ExecLine(m.context, msg)
	//case common.ToggleHelpMsg:
	//	if m.stacked == nil {
	//		m.stacked = helppage.New(m.context)
	//		if p, ok := m.stacked.(common.ISizeable); ok {
	//			p.SetHeight(m.Height - 2)
	//			p.SetWidth(m.Width)
	//		}
	//	} else {
	//		m.stacked = nil
	//	}
	//	return m, nil
	//case common.ShowDiffMsg:
	//	m.diff = diff.New(string(msg), m.Width, m.Height)
	//	return m, m.diff.Init()
	//case common.UpdateRevisionsSuccessMsg:
	//	m.state = common.Ready
	//case triggerAutoRefreshMsg:
	//	return m, tea.Batch(m.scheduleAutoRefresh(), func() tea.Msg {
	//		return common.AutoRefreshMsg{}
	//	})
	//case common.UpdateRevSetMsg:
	//	m.context.CurrentRevset = string(msg)
	//	m.revsetModel.AddToHistory(m.context.CurrentRevset)
	//	return m, common.Refresh
	//case common.ShowPreview:
	//	m.context.Preview.SetVisible(bool(msg))
	//	return m, tea.Batch(cmds...)
	//}
	//
	//if m.revsetModel.Editing {
	//	m.revsetModel.Model, cmd = m.revsetModel.Update(msg)
	//	cmds = append(cmds, cmd)
	//}
	//
	//m.status.Model, cmd = m.status.Update(msg)
	//cmds = append(cmds, cmd)
	//
	//m.flash, cmd = m.flash.Update(msg)
	//cmds = append(cmds, cmd)
	//
	//if m.stacked != nil {
	//	m.stacked, cmd = m.stacked.Update(msg)
	//	cmds = append(cmds, cmd)
	//}
	//
	//if m.oplog != nil {
	//	m.oplog, cmd = m.oplog.Update(msg)
	//	cmds = append(cmds, cmd)
	//} else {
	//	m.revisions.Model, cmd = m.revisions.Update(msg)
	//	cmds = append(cmds, cmd)
	//}
	//
	//if m.context.Preview.Visible {
	//	m.previewModel.Model, cmd = m.previewModel.Update(msg)
	//	cmds = append(cmds, cmd)
	//}
	//
	//return m, tea.Batch(cmds...)
}

func (m Model) updateStatus() {
	switch {
	case m.diff != nil:
		//m.status.SetMode("diff")
		//m.status.SetHelp(m.diff)
	//case m.oplog != nil:
	//m.status.SetMode("oplog")
	//m.status.SetHelp(m.oplog)
	case m.stacked != nil:
		//if s, ok := m.stacked.(help.KeyMap); ok {
		//	m.status.SetHelp(s)
		//}
	case m.leader != nil:
		//m.status.SetMode("leader")
		//m.status.SetHelp(m.leader)
	default:
		//m.status.SetHelp(m.revisions)
		//m.status.SetMode(m.revisions.CurrentOperation().Name())
	}
}

func (m Model) View() string {
	m.updateStatus()
	m.BaseView.Layout()
	return m.BaseView.View()
	//status := m.Top[0]
	//preview := m.Right[0]
	//top := m.Top[len(m.Top)-1]
	//footer := status.View()
	//footerHeight := lipgloss.Height(footer)
	//
	//if m.diff != nil {
	//	m.diff.SetHeight(m.Height - footerHeight)
	//	return lipgloss.JoinVertical(0, m.diff.View(), footer)
	//}
	//
	//topView := top.View()
	//topViewHeight := lipgloss.Height(topView)
	//
	//bottomPreviewHeight := 0
	//if m.context.Preview.Visible && m.context.Preview.AtBottom {
	//	bottomPreviewHeight = int(float64(m.Height) * (m.context.Preview.WindowPercentage / 100.0))
	//}
	//leftView := m.renderLeftView(footerHeight, topViewHeight, bottomPreviewHeight)
	//centerView := leftView
	//
	//if m.context.Preview.Visible {
	//	if m.context.Preview.AtBottom {
	//		preview.SetWidth(m.Width)
	//		preview.SetHeight(bottomPreviewHeight)
	//	} else {
	//		preview.SetWidth(m.Width - lipgloss.Width(leftView))
	//		preview.SetHeight(m.Height - footerHeight - topViewHeight)
	//	}
	//	previewView := preview.View()
	//	if m.context.Preview.AtBottom {
	//		centerView = lipgloss.JoinVertical(lipgloss.Top, leftView, previewView)
	//	} else {
	//		centerView = lipgloss.JoinHorizontal(lipgloss.Left, leftView, previewView)
	//	}
	//}
	//
	//if m.stacked != nil {
	//	stackedView := m.stacked.View()
	//	w, h := lipgloss.Size(stackedView)
	//	sx := (m.Width - w) / 2
	//	sy := (m.Height - h) / 2
	//	centerView = screen.Stacked(centerView, stackedView, sx, sy)
	//}
	//
	//full := lipgloss.JoinVertical(0, topView, centerView, footer)
	//flashMessageView := m.flash.View()
	//if flashMessageView != "" {
	//	mw, mh := lipgloss.Size(flashMessageView)
	//	full = screen.Stacked(full, flashMessageView, m.Width-mw, m.Height-mh-1)
	//}
	//statusFuzzyView := m.status.FuzzyView()
	//if statusFuzzyView != "" {
	//	_, mh := lipgloss.Size(statusFuzzyView)
	//	full = screen.Stacked(full, statusFuzzyView, 0, m.Height-mh-1)
	//}
	//return full
}

//func (m Model) renderLeftView(footerHeight int, topViewHeight int, bottomPreviewHeight int) string {
//	leftView := ""
//	w := m.BaseView.Width
//	h := 0
//
//	if m.context.Preview.Visible {
//		if m.context.Preview.AtBottom {
//			h = bottomPreviewHeight
//		} else {
//			w = m.Width - int(float64(m.Width)*(m.context.Preview.WindowPercentage/100.0))
//		}
//	}
//
//	left := m.Middle[len(m.Middle)-1]
//	if left == nil {
//		return leftView
//	}
//
//	left.SetWidth(w)
//	left.SetHeight(m.Height - footerHeight - topViewHeight - h)
//	leftView = left.View()
//	return leftView
//}

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
	//if m.oplog != nil {
	//	return false
	//}
	//if m.revisions.Focused && m.revisions.Visible {
	//	return true
	//}
	//if m.revisions.CurrentOperation().Name() == "normal" {
	//	return true
	//}
	return true
}

func New(c *context.MainContext) tea.Model {
	size := &common.Sizeable{Width: 80, Height: 24}

	// Create the stacked container for revisions view
	stackedContainer := view.NewStackedLayoutContainer("stacked-revisions")
	revisionsView := revisions.New(c)
	stackedContainer.Add(revisionsView, view.Grow(1))

	// Create horizontal container with the stacked container
	horizontalContainer := view.HorizontalContainer("main")
	horizontalContainer.Add(stackedContainer.BaseView, view.Grow(1))

	// Create the root vertical container
	root := view.VerticalContainer("root")
	root.Add(revset.New(c), view.FitContent())
	root.AddContainer(horizontalContainer, view.Grow(1))
	root.Add(status.New(c), view.FitContent())

	// Set initial size
	root.Sizeable = size
	c.Root = root

	m := Model{
		BaseView:            root.BaseView,
		stackedContainer:    stackedContainer,
		horizontalContainer: horizontalContainer,
		context:             c,
		keyMap:              config.Current.GetKeyMap(),
		state:               common.Loading,
		flash:               flash.New(c),
	}
	return m
}
