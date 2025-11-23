package ui

import (
	"time"

	"charm.land/bubbles/v2/help"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/idursun/jjui/internal/ui/flash"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
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
	stacked      common.Stackable
}

type triggerAutoRefreshMsg struct{}

func (m Model) Init() tea.Cmd {
	//return tea.Batch(tea.SetWindowTitle(fmt.Sprintf("jjui - %s", m.context.Location)), m.revisions.Init(), m.scheduleAutoRefresh())
	return tea.Batch(m.revisions.Init(), m.scheduleAutoRefresh())
}

func (m Model) handleFocusInputMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
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

	if m.leader != nil {
		m.leader, cmd = m.leader.Update(msg)
		return m, cmd, true
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
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

		if m.revisions.IsEditing() {
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
	case tea.FocusMsg:
		return m, common.RefreshAndKeepSelections
	case tea.KeyPressMsg:
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
			m.oplog = oplog.New(m.context, m.Width, m.Height)
			return m, m.oplog.Init()
		case key.Matches(msg, m.keyMap.Revset) && m.revisions.InNormalMode():
			m.revsetModel, _ = m.revsetModel.Update(revset.EditRevSetMsg{Clear: m.state != common.Error})
			return m, nil
		case key.Matches(msg, m.keyMap.Git.Mode) && m.revisions.InNormalMode():
			m.stacked = git.NewModel(m.context, m.revisions.SelectedRevisions(), m.Width, m.Height)
			return m, m.stacked.Init()
		case key.Matches(msg, m.keyMap.Undo) && m.revisions.InNormalMode():
			m.stacked = undo.NewModel(m.context)
			cmds = append(cmds, m.stacked.Init())
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Redo) && m.revisions.InNormalMode():
			m.stacked = redo.NewModel(m.context)
			cmds = append(cmds, m.stacked.Init())
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Bookmark.Mode) && m.revisions.InNormalMode():
			changeIds := m.revisions.GetCommitIds()
			m.stacked = bookmarks.NewModel(m.context, m.revisions.SelectedRevision(), changeIds, m.Width, m.Height)
			cmds = append(cmds, m.stacked.Init())
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Help):
			cmds = append(cmds, common.ToggleHelp)
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Mode, m.keyMap.Preview.ToggleBottom):
			if key.Matches(msg, m.keyMap.Preview.ToggleBottom) {
				previewPos := m.previewModel.AtBottom()
				m.previewModel.SetPosition(false, !previewPos)
				if m.previewModel.Visible() {
					return m, tea.Batch(cmds...)
				}
			}
			m.previewModel.ToggleVisible()
			cmds = append(cmds, common.SelectionChanged)
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Expand) && m.previewModel.Visible():
			m.previewModel.Expand()
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Preview.Shrink) && m.previewModel.Visible():
			m.previewModel.Shrink()
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.CustomCommands):
			m.stacked = customcommands.NewModel(m.context, m.Width, m.Height)
			cmds = append(cmds, m.stacked.Init())
			return m, tea.Batch(cmds...)
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
			return m, common.FileSearch(m.context.CurrentRevset, m.previewModel.Visible(), rev, out)
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
			if p, ok := m.stacked.(common.ISizeable); ok {
				p.SetHeight(m.Height - 2)
				p.SetWidth(m.Width)
			}
		} else {
			m.stacked = nil
		}
		return m, nil
	case common.ShowDiffMsg:
		m.diff = diff.New(string(msg), m.Width, m.Height)
		return m, m.diff.Init()
	case common.UpdateRevisionsSuccessMsg:
		m.state = common.Ready
	case triggerAutoRefreshMsg:
		return m, tea.Batch(m.scheduleAutoRefresh(), func() tea.Msg {
			return common.AutoRefreshMsg{}
		})
	case common.UpdateRevSetMsg:
		m.context.CurrentRevset = string(msg)
		m.revsetModel.AddToHistory(m.context.CurrentRevset)
		return m, common.Refresh
	case common.ShowPreview:
		m.previewModel.SetVisible(bool(msg))
		cmds = append(cmds, common.SelectionChanged)
		return m, tea.Batch(cmds...)
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
		m.revsetModel, cmd = m.revsetModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.status, cmd = m.status.Update(msg)
	cmds = append(cmds, cmd)

	m.flash, cmd = m.flash.Update(msg)
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

	if m.previewModel.Visible() {
		m.previewModel, cmd = m.previewModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateStatus() {
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

func (m Model) UpdatePreviewPosition() {
	if m.previewModel.AutoPosition() {
		atBottom := m.Height >= m.Width/2
		m.previewModel.SetPosition(true, atBottom)
	}
}

func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	v.ReportFocus = true

	m.updateStatus()
	footer := m.status.View()
	var centerArea uv.Rectangle
	var footerArea uv.Rectangle
	var topArea uv.Rectangle

	area := uv.Rect(0, 0, m.Width, m.Height)
	if m.diff != nil {
		m.diff.SetHeight(m.Height - footer.Bounds().Dy())
		centerArea, footerArea = uv.SplitVertical(area, uv.Fixed(m.Height-footer.Bounds().Dy()))
		c := lipgloss.NewCanvas(m.Width, m.Height)
		c.Compose(m.diff.View().X(centerArea.Min.X).Y(centerArea.Min.Y))
		c.Compose(footer.X(footerArea.Min.X).Y(footerArea.Min.Y))
		v.SetContent(c)
		return v
	}

	topView := m.revsetModel.View(uv.Rect(0, 0, m.Width, 2))
	topViewHeight := topView.Bounds().Dy()
	topArea, centerArea = uv.SplitVertical(area, uv.Fixed(topViewHeight))
	centerArea, footerArea = uv.SplitVertical(centerArea, uv.Fixed(centerArea.Bounds().Dy()-footer.Bounds().Dy()))

	revisionsArea := centerArea
	var previewArea uv.Rectangle

	m.UpdatePreviewPosition()

	var previewLayer *lipgloss.Layer
	if m.previewModel.Visible() {
		if m.previewModel.AtBottom() {
			revisionsArea, previewArea = uv.SplitVertical(centerArea, uv.Percent(100-m.previewModel.WindowPercentage()))
		} else {
			revisionsArea, previewArea = uv.SplitHorizontal(centerArea, uv.Percent(100-m.previewModel.WindowPercentage()))
		}
		m.previewModel.SetWidth(previewArea.Dx())
		m.previewModel.SetHeight(previewArea.Dy())
		previewLayer = m.previewModel.View().X(previewArea.Min.X).Y(previewArea.Min.Y)
	}
	var centerView *lipgloss.Layer
	if m.oplog != nil {
		centerView = m.oplog.View(revisionsArea)
	} else {
		centerView = m.revisions.View(revisionsArea)
	}

	centerView = centerView.X(centerArea.Min.X).Y(centerArea.Min.Y)
	topView = topView.X(topArea.Min.X).Y(topArea.Min.Y)
	footer = footer.X(footerArea.Min.X).Y(footerArea.Min.Y)
	c := lipgloss.NewCanvas(m.Width, m.Height)
	c.Compose(topView)
	c.Compose(centerView)
	if previewLayer != nil {
		c.Compose(previewLayer)
	}
	c.Compose(footer)

	if m.stacked != nil {
		stackedView := m.stacked.View()
		w, h := stackedView.Bounds().Dx(), stackedView.Bounds().Dy()
		sx := (m.Width - w) / 2
		sy := (m.Height - h) / 2

		c.Compose(stackedView.X(sx).Y(sy).Z(10))
	}

	flashMessages := m.flash.View(m.Width, m.Height)
	for i := range flashMessages {
		c.Compose(flashMessages[i])
	}
	statusFuzzyView := m.status.FuzzyView()
	if statusFuzzyView != nil {
		mh := statusFuzzyView.Bounds().Dy()
		c.Compose(statusFuzzyView.Y(m.Height - mh - 1).Z(5))
	}
	v.SetContent(c)
	return v
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
