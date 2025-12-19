package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/scripting"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/password"

	"github.com/idursun/jjui/internal/ui/flash"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/bookmarks"
	"github.com/idursun/jjui/internal/ui/choose"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	customcommands "github.com/idursun/jjui/internal/ui/custom_commands"
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/exec_process"
	"github.com/idursun/jjui/internal/ui/git"
	"github.com/idursun/jjui/internal/ui/helppage"
	"github.com/idursun/jjui/internal/ui/input"
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

type SizableModel interface {
	common.Model
	common.IViewNode
}

type Model struct {
	*common.ViewNode
	revisions       *revisions.Model
	oplog           *oplog.Model
	revsetModel     *revset.Model
	previewModel    *preview.Model
	diff            *diff.Model
	leader          *leader.Model
	flash           *flash.Model
	state           common.State
	status          *status.Model
	password        *password.Model
	context         *context.MainContext
	scriptRunner    *scripting.Runner
	keyMap          config.KeyMappings[key.Binding]
	stacked         SizableModel
	dragTarget      common.Draggable
	sequenceOverlay *customcommands.SequenceOverlay
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
		if m.password != nil {
			return m.password.Update(msg), true
		}

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

func (m *Model) handleCustomCommandSequence(msg tea.KeyMsg) tea.Cmd {
	if !m.ensureSequenceOverlay(msg) {
		return nil
	}

	res := m.sequenceOverlay.HandleKey(msg)
	if !res.Active {
		m.sequenceOverlay = nil
	}
	if res.Cmd != nil {
		return res.Cmd
	}
	return nil
}

func (m *Model) ensureSequenceOverlay(msg tea.KeyMsg) bool {
	if m.sequenceOverlay != nil {
		return true
	}
	if !m.shouldStartSequenceOverlay(msg) {
		return false
	}
	m.sequenceOverlay = customcommands.NewSequenceOverlay(m.context)
	m.sequenceOverlay.Parent = m.ViewNode
	return true
}

func (m *Model) shouldStartSequenceOverlay(msg tea.KeyMsg) bool {
	for _, command := range customcommands.SortedCustomCommands(m.context) {
		seq := command.Sequence()
		if len(seq) == 0 || !command.IsApplicableTo(m.context.SelectedItem) {
			continue
		}
		if key.Matches(msg, seq[0]) {
			return true
		}
	}
	return false
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if cmd, handled := m.handleFocusInputMessage(msg); handled {
		return cmd
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case common.DeferredUpdateMsg:
		if msg.Fn != nil {
			return msg.Fn()
		}
		return nil
	case tea.FocusMsg:
		return tea.Batch(common.RefreshAndKeepSelections, tea.EnableMouseCellMotion)
	case tea.MouseMsg:
		if m.stacked != nil {
			// for now, stacked windows don't respond to mouse events
			return nil
		}
		if m.dragTarget != nil && m.dragTarget.IsDragging() {
			switch msg.Action {
			case tea.MouseActionRelease:
				cmd := m.dragTarget.DragEnd(msg.X, msg.Y)
				m.dragTarget = nil
				return cmd
			case tea.MouseActionMotion:
				return m.dragTarget.DragMove(msg.X, msg.Y)
			}
		} else if m.dragTarget != nil && !m.dragTarget.IsDragging() {
			m.dragTarget = nil
		}

		if model := m.findViewAt(msg.X, msg.Y); model != nil {
			if draggable, ok := model.(common.Draggable); ok && msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
				if draggable.DragStart(msg.X, msg.Y) {
					m.dragTarget = draggable
					return nil
				}
			}
			return model.Update(msg)
		}
		return nil
	case tea.KeyMsg:
		// Forward all key presses to the custom sequence handler first.
		wasPartialSequenceMatch := m.sequenceOverlay != nil
		if cmd := m.handleCustomCommandSequence(msg); cmd != nil || m.sequenceOverlay != nil {
			return cmd
		}
		if wasPartialSequenceMatch {
			// If we were in a partial sequence but the key didn't match, don't
			// process it further.
			return nil
		}

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
			m.oplog = oplog.New(m.context)
			m.oplog.Parent = m.ViewNode
			return m.oplog.Init()
		case key.Matches(msg, m.keyMap.Revset) && m.revisions.InNormalMode():
			return m.revsetModel.Update(intents.Edit{Clear: m.state != common.Error})
		case key.Matches(msg, m.keyMap.Git.Mode) && m.revisions.InNormalMode():
			model := git.NewModel(m.context, m.revisions.SelectedRevisions())
			model.Parent = m.ViewNode
			m.stacked = model
			return m.stacked.Init()
		case key.Matches(msg, m.keyMap.Undo) && m.revisions.InNormalMode():
			model := undo.NewModel(m.context)
			model.Parent = m.ViewNode
			m.stacked = model
			cmds = append(cmds, m.stacked.Init())
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Redo) && m.revisions.InNormalMode():
			model := redo.NewModel(m.context)
			model.Parent = m.ViewNode
			m.stacked = model
			cmds = append(cmds, m.stacked.Init())
			return tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Bookmark.Mode) && m.revisions.InNormalMode():
			changeIds := m.revisions.GetCommitIds()
			model := bookmarks.NewModel(m.context, m.revisions.SelectedRevision(), changeIds)
			model.Parent = m.ViewNode
			m.stacked = model
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
			model := customcommands.NewModel(m.context)
			model.Parent = m.ViewNode
			m.stacked = model
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
			for _, command := range customcommands.SortedCustomCommands(m.context) {
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
	case common.ExecProcessCompletedMsg:
		cmds = append(cmds, common.Refresh)
	case common.ToggleHelpMsg:
		if m.stacked == nil {
			h := helppage.New(m.context)
			h.Parent = m.ViewNode
			m.stacked = h
		} else {
			m.stacked = nil
		}
		return nil
	case common.ShowDiffMsg:
		m.diff = diff.New(string(msg))
		return m.diff.Init()
	case common.UpdateRevisionsSuccessMsg:
		m.state = common.Ready
	case customcommands.SequenceTimeoutMsg:
		if m.sequenceOverlay == nil {
			return nil
		}
		res := m.sequenceOverlay.HandleTimeout(msg)
		if !res.Active {
			m.sequenceOverlay = nil
		}
		return res.Cmd
	case triggerAutoRefreshMsg:
		return tea.Batch(m.scheduleAutoRefresh(), func() tea.Msg {
			return common.AutoRefreshMsg{}
		})
	case common.UpdateRevSetMsg:
		m.context.CurrentRevset = string(msg)
		if m.context.CurrentRevset == "" {
			m.context.CurrentRevset = m.context.DefaultRevset
		}
		m.revsetModel.AddToHistory(m.context.CurrentRevset)
		return common.Refresh
	case common.RunLuaScriptMsg:
		runner, cmd, err := scripting.RunScript(m.context, msg.Script)
		if err != nil {
			return func() tea.Msg {
				return common.CommandCompletedMsg{Err: err}
			}
		}
		m.scriptRunner = runner
		if cmd == nil && (runner == nil || runner.Done()) {
			m.scriptRunner = nil
		}
		return cmd
	case common.ShowChooseMsg:
		model := choose.NewWithTitle(msg.Options, msg.Title)
		model.Parent = m.ViewNode
		m.stacked = model
		return m.stacked.Init()
	case choose.SelectedMsg, choose.CancelledMsg:
		m.stacked = nil
	case common.ShowInputMsg:
		model := input.NewWithTitle(msg.Title, msg.Prompt)
		model.Parent = m.ViewNode
		m.stacked = model
		return m.stacked.Init()
	case input.SelectedMsg, input.CancelledMsg:
		m.stacked = nil
	case common.ShowPreview:
		m.previewModel.SetVisible(bool(msg))
		cmds = append(cmds, common.SelectionChanged)
		return tea.Batch(cmds...)
	case common.TogglePasswordMsg:
		if m.password != nil {
			// let the current prompt clean itself
			m.password.Update(msg)
		}
		if msg.Password == nil {
			m.password = nil
		} else {
			// overwrite current prompt. This can happen for ssh-sk keys:
			//   - first prompt reads "Confirm user presence for ..."
			//   - if the user denies the request on the device, a new prompt automatically happen "Enter PIN for ...
			m.password = password.New(msg, m.ViewNode)
		}

	case tea.WindowSizeMsg:
		m.SetFrame(cellbuf.Rect(0, 0, msg.Width, msg.Height))
		m.flash.SetWidth(m.Width)
		m.flash.SetHeight(m.Height)
	}

	cmds = append(cmds, m.revsetModel.Update(msg))
	cmds = append(cmds, m.status.Update(msg))
	cmds = append(cmds, m.flash.Update(msg))

	if m.stacked != nil {
		cmds = append(cmds, m.stacked.Update(msg))
	}

	if m.scriptRunner != nil {
		if cmd := m.scriptRunner.HandleMsg(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
		if m.scriptRunner.Done() {
			m.scriptRunner = nil
		}
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
	if m.Width == 0 || m.Height == 0 {
		return ""
	}
	m.updateStatus()
	m.status.SetWidth(m.Width)
	footer := m.status.View()
	footerHeight := lipgloss.Height(footer)

	if m.diff != nil {
		m.diff.SetFrame(cellbuf.Rect(0, footerHeight, m.Width, m.Height-footerHeight))
		return lipgloss.JoinVertical(0, m.diff.View(), footer)
	}

	screenBuf := cellbuf.NewBuffer(m.Width, m.Height)
	centerArea := cellbuf.Rect(0, 0, m.Width, m.Height)
	var topArea, previewArea, bottomArea cellbuf.Rectangle

	topView := m.revsetModel.View()
	topViewHeight := lipgloss.Height(topView)
	topArea, centerArea = layout.SplitVertical(centerArea, layout.Fixed(topViewHeight))
	cellbuf.SetContentRect(screenBuf, topView, topArea)

	centerArea, bottomArea = layout.SplitVertical(centerArea, layout.Fixed(centerArea.Dy()-footerHeight))
	cellbuf.SetContentRect(screenBuf, footer, bottomArea)

	if m.previewModel.Visible() {
		m.UpdatePreviewPosition()
		if m.previewModel.AtBottom() {
			centerArea, previewArea = layout.SplitVertical(centerArea, layout.Percent(100-m.previewModel.WindowPercentage()))
		} else {
			centerArea, previewArea = layout.SplitHorizontal(centerArea, layout.Percent(100-m.previewModel.WindowPercentage()))
		}
		m.previewModel.SetFrame(previewArea)
	}

	var leftView string
	if m.oplog != nil {
		m.oplog.SetFrame(centerArea)
		leftView = m.oplog.View()
	} else {
		m.revisions.SetFrame(centerArea)
		leftView = m.revisions.View()
	}

	cellbuf.SetContentRect(screenBuf, leftView, centerArea)
	if m.previewModel.Visible() {
		cellbuf.SetContentRect(screenBuf, m.previewModel.View(), previewArea)
	}

	if m.stacked != nil {
		stackedView := m.stacked.View()
		cellbuf.SetContentRect(screenBuf, stackedView, m.stacked.GetViewNode().Frame)
	}

	if m.sequenceOverlay != nil {
		m.sequenceOverlay.Parent = m.ViewNode
		view := m.sequenceOverlay.View()
		cellbuf.SetContentRect(screenBuf, view, m.sequenceOverlay.Frame)
	}

	flashMessageView := m.flash.View()
	if flashMessageView != nil {
		for _, v := range flashMessageView {
			cellbuf.SetContentRect(screenBuf, v.Content, v.Rect)
		}
	}
	statusFuzzyView := m.status.FuzzyView()
	if statusFuzzyView != "" {
		_, mh := lipgloss.Size(statusFuzzyView)
		cellbuf.SetContentRect(screenBuf, statusFuzzyView, cellbuf.Rect(0, m.Height-mh-1, m.Width, mh))
	}

	if m.password != nil {
		view := m.password.View()
		cellbuf.SetContentRect(screenBuf, view, m.password.Frame)
	}

	finalView := cellbuf.Render(screenBuf)
	return strings.ReplaceAll(finalView, "\r", "")
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

func (m *Model) findViewAt(x, y int) common.IMouseAware {
	// well, these are all the views that can receive mouse input for now
	pt := cellbuf.Pos(x, y)
	if m.diff != nil && pt.In(m.diff.Frame) {
		return m.diff
	}
	if m.oplog != nil && pt.In(m.oplog.Frame) {
		return m.oplog
	}
	if m.oplog == nil && pt.In(m.revisions.Frame) {
		return m.revisions
	}
	if m.previewModel.Visible() && pt.In(m.previewModel.Frame) {
		return m.previewModel
	}
	return nil
}

var _ tea.Model = (*wrapper)(nil)

type (
	frameTickMsg struct{}
	wrapper      struct {
		ui                 *Model
		scheduledNextFrame bool
		render             bool
		cachedFrame        string
	}
)

func (w *wrapper) Init() tea.Cmd {
	return w.ui.Init()
}

func (w *wrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(frameTickMsg); ok {
		w.render = true
		w.scheduledNextFrame = false
		return w, nil
	}
	var cmd tea.Cmd
	cmd = w.ui.Update(msg)
	if !w.scheduledNextFrame {
		w.scheduledNextFrame = true
		return w, tea.Batch(cmd, tea.Tick(time.Millisecond*8, func(t time.Time) tea.Msg {
			return frameTickMsg{}
		}))
	}
	return w, cmd
}

func (w *wrapper) View() string {
	if w.render {
		w.cachedFrame = w.ui.View()
		w.render = false
	}
	return w.cachedFrame
}

func NewUI(c *context.MainContext) *Model {
	frame := common.NewViewNode(0, 0)

	revisionsModel := revisions.New(c)
	revisionsModel.Parent = frame

	statusModel := status.New(c)
	statusModel.Parent = frame

	flashView := flash.New(c)
	flashView.Parent = frame

	previewModel := preview.New(c)
	previewModel.Parent = frame

	revsetModel := revset.New(c)
	revsetModel.Parent = frame

	return &Model{
		ViewNode:     frame,
		context:      c,
		keyMap:       config.Current.GetKeyMap(),
		state:        common.Loading,
		revisions:    revisionsModel,
		previewModel: previewModel,
		status:       statusModel,
		revsetModel:  revsetModel,
		flash:        flashView,
	}
}

func New(c *context.MainContext) tea.Model {
	return &wrapper{ui: NewUI(c)}
}
