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
	"github.com/idursun/jjui/internal/ui/render"

	"github.com/idursun/jjui/internal/ui/flash"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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

type Model struct {
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
	stacked         common.ImmediateModel
	sequenceOverlay *customcommands.SequenceOverlay
	displayContext  *render.DisplayContext
	width           int
	height          int
	revisionsSplit  *split
	activeSplit     *split
	splitActive     bool
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
			return common.SelectionChanged(m.context.SelectedItem), true
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
	case tea.FocusMsg:
		return tea.Batch(common.RefreshAndKeepSelections, tea.EnableMouseCellMotion)
	case tea.MouseMsg:
		if m.splitActive {
			switch msg.Action {
			case tea.MouseActionRelease:
				m.splitActive = false
			case tea.MouseActionMotion:
				if m.activeSplit != nil {
					m.activeSplit.DragTo(msg.X, msg.Y)
				}
				return nil
			}
		}

		// Process interactions from DisplayContext first
		if m.displayContext != nil {
			if interactionMsg, handled := m.displayContext.ProcessMouseEvent(msg); handled {
				if interactionMsg != nil {
					// Send the interaction message back through Update
					return func() tea.Msg { return interactionMsg }
				}
				return nil
			}
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

		if key.Matches(msg, m.keyMap.Cancel) && (m.state == common.Error || m.stacked != nil || m.flash.Any()) {
			return m.handleIntent(intents.Cancel{})
		}

		switch {
		case key.Matches(msg, m.keyMap.Quit) && m.isSafeToQuit():
			return m.handleIntent(intents.Quit{})
		case key.Matches(msg, m.keyMap.OpLog.Mode) && m.revisions.InNormalMode():
			return m.handleIntent(intents.OpLogOpen{})
		case key.Matches(msg, m.keyMap.Revset) && m.revisions.InNormalMode():
			return m.handleIntent(intents.Edit{Clear: m.state != common.Error})
		case key.Matches(msg, m.keyMap.Git.Mode) && m.revisions.InNormalMode():
			return m.handleIntent(intents.OpenGit{})
		case key.Matches(msg, m.keyMap.Undo) && m.revisions.InNormalMode():
			return m.handleIntent(intents.Undo{})
		case key.Matches(msg, m.keyMap.Redo) && m.revisions.InNormalMode():
			return m.handleIntent(intents.Redo{})
		case key.Matches(msg, m.keyMap.Bookmark.Mode) && m.revisions.InNormalMode():
			return m.handleIntent(intents.OpenBookmarks{})
		case key.Matches(msg, m.keyMap.Help):
			return m.handleIntent(intents.HelpToggle{})
		case key.Matches(msg, m.keyMap.Preview.Mode):
			return m.handleIntent(intents.PreviewToggle{})
		case key.Matches(msg, m.keyMap.Preview.ToggleBottom):
			return m.handleIntent(intents.PreviewToggleBottom{})
		case key.Matches(msg, m.keyMap.Preview.Expand) && m.previewModel.Visible():
			return m.handleIntent(intents.PreviewExpand{})
		case key.Matches(msg, m.keyMap.Preview.Shrink) && m.previewModel.Visible():
			return m.handleIntent(intents.PreviewShrink{})
		case m.previewModel.Visible() && key.Matches(msg,
			m.keyMap.Preview.HalfPageUp,
			m.keyMap.Preview.HalfPageDown,
			m.keyMap.Preview.ScrollUp,
			m.keyMap.Preview.ScrollDown):
			switch {
			case key.Matches(msg, m.keyMap.Preview.HalfPageUp):
				return m.handleIntent(intents.PreviewScroll{Kind: intents.PreviewHalfPageUp})
			case key.Matches(msg, m.keyMap.Preview.HalfPageDown):
				return m.handleIntent(intents.PreviewScroll{Kind: intents.PreviewHalfPageDown})
			case key.Matches(msg, m.keyMap.Preview.ScrollUp):
				return m.handleIntent(intents.PreviewScroll{Kind: intents.PreviewScrollUp})
			case key.Matches(msg, m.keyMap.Preview.ScrollDown):
				return m.handleIntent(intents.PreviewScroll{Kind: intents.PreviewScrollDown})
			}
		case key.Matches(msg, m.keyMap.CustomCommands):
			return m.handleIntent(intents.OpenCustomCommands{})
		case key.Matches(msg, m.keyMap.Leader):
			return m.handleIntent(intents.OpenLeader{})
		case key.Matches(msg, m.keyMap.FileSearch.Toggle):
			return m.handleIntent(intents.FileSearchToggle{})
		case key.Matches(msg, m.keyMap.ExecJJ) && m.revisions.InNormalMode():
			return m.handleIntent(intents.ExecJJ{})
		case key.Matches(msg, m.keyMap.ExecShell) && m.revisions.InNormalMode():
			return m.handleIntent(intents.ExecShell{})
		case key.Matches(msg, m.keyMap.QuickSearch) && m.revisions.InNormalMode():
			return m.handleIntent(intents.QuickSearch{})
		case key.Matches(msg, m.keyMap.Suspend):
			return m.handleIntent(intents.Suspend{})
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
	case intents.Intent:
		if cmd := m.handleIntent(msg); cmd != nil {
			return cmd
		}
	case common.ExecMsg:
		return exec_process.ExecLine(m.context, msg)
	case common.ExecProcessCompletedMsg:
		cmds = append(cmds, common.Refresh)
	case common.ToggleHelpMsg:
		if m.stacked == nil {
			h := helppage.New(m.context)
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
		m.stacked = model
		return m.stacked.Init()
	case choose.SelectedMsg, choose.CancelledMsg:
		m.stacked = nil
	case common.ShowInputMsg:
		model := input.NewWithTitle(msg.Title, msg.Prompt)
		m.stacked = model
		return m.stacked.Init()
	case input.SelectedMsg, input.CancelledMsg:
		m.stacked = nil
	case common.ShowPreview:
		m.previewModel.SetVisible(bool(msg))
		cmds = append(cmds, common.SelectionChanged(m.context.SelectedItem))
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
			m.password = password.New(msg)
		}
	case SplitDragMsg:
		m.activeSplit = msg.Split
		m.splitActive = true
		if m.activeSplit != nil {
			m.activeSplit.DragTo(msg.X, msg.Y)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Unhandled key messages go to the main view (oplog or revisions)
	// Other messages are broadcast to all models
	if common.IsInputMessage(msg) {
		if m.oplog != nil {
			cmds = append(cmds, m.oplog.Update(msg))
		} else {
			cmds = append(cmds, m.revisions.Update(msg))
		}
		return tea.Batch(cmds...)
	}

	cmds = append(cmds, m.revsetModel.Update(msg))
	cmds = append(cmds, m.status.Update(msg))
	cmds = append(cmds, m.flash.Update(msg))
	if m.diff != nil {
		cmds = append(cmds, m.diff.Update(msg))
	}

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
		atBottom := m.height >= m.width/2
		m.previewModel.SetPosition(true, atBottom)
	}
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	m.displayContext = render.NewDisplayContext()

	m.updateStatus()

	box := layout.NewBox(cellbuf.Rect(0, 0, m.width, m.height))
	screenBuf := cellbuf.NewBuffer(m.width, m.height)

	if m.diff != nil {
		m.renderDiffLayout(box)
		m.displayContext.Render(screenBuf)
		content := cellbuf.Render(screenBuf)
		return strings.ReplaceAll(content, "\r", "")
	}

	if m.previewModel.Visible() {
		m.UpdatePreviewPosition()
	}
	m.syncPreviewSplitOrientation()
	if m.oplog != nil {
		m.renderOpLogLayout(box)
	} else {
		m.renderRevisionsLayout(box)
	}

	if m.stacked != nil {
		m.stacked.ViewRect(m.displayContext, box)
	}

	if m.sequenceOverlay != nil {
		m.sequenceOverlay.ViewRect(m.displayContext, box)
	}

	m.flash.ViewRect(m.displayContext, box)

	if m.password != nil {
		m.password.ViewRect(m.displayContext, box)
	}

	m.displayContext.Render(screenBuf)
	finalView := cellbuf.Render(screenBuf)
	return strings.ReplaceAll(finalView, "\r", "")
}

func (m *Model) renderDiffLayout(box layout.Box) {
	m.renderWithStatus(box, func(content layout.Box) {
		m.diff.ViewRect(m.displayContext, content)
	})
}

func (m *Model) renderOpLogLayout(box layout.Box) {
	m.renderWithStatus(box, func(content layout.Box) {
		m.renderSplit(m.oplog, content)
	})
}

func (m *Model) renderRevisionsLayout(box layout.Box) {
	rows := box.V(layout.Fixed(1), layout.Fill(1), layout.Fixed(1))
	if len(rows) < 3 {
		return
	}
	m.revsetModel.ViewRect(m.displayContext, rows[0])
	m.renderSplit(m.revisions, rows[1])
	m.status.ViewRect(m.displayContext, rows[2])
}

func (m *Model) renderWithStatus(box layout.Box, renderContent func(layout.Box)) {
	rows := box.V(layout.Fill(1), layout.Fixed(1))
	if len(rows) < 2 {
		return
	}
	renderContent(rows[0])
	m.status.ViewRect(m.displayContext, rows[1])
}

func (m *Model) renderSplit(primary common.ImmediateModel, box layout.Box) {
	if m.revisionsSplit == nil {
		return
	}
	m.revisionsSplit.Primary = primary
	m.revisionsSplit.Secondary = m.previewModel
	m.revisionsSplit.Render(m.displayContext, box)
}

func (m *Model) syncPreviewSplitOrientation() {
	if m.revisionsSplit == nil {
		return
	}
	vertical := m.previewModel.AtBottom()
	m.revisionsSplit.Vertical = vertical
}

func (m *Model) initSplit() {
	splitState := newSplitState(config.Current.Preview.WidthPercentage)

	m.revisionsSplit = newSplit(
		splitState,
		m.revisions,
		m.previewModel,
	)
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

func (m *Model) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent := intent.(type) {
	case intents.Cancel:
		switch {
		case m.state == common.Error:
			m.state = common.Ready
		case m.stacked != nil:
			m.stacked = nil
		case m.flash.Any():
			m.flash.DeleteOldest()
		}
		return nil
	case intents.Undo:
		if !m.revisions.InNormalMode() {
			return nil
		}
		model := undo.NewModel(m.context)
		m.stacked = model
		return m.stacked.Init()
	case intents.Redo:
		if !m.revisions.InNormalMode() {
			return nil
		}
		model := redo.NewModel(m.context)
		m.stacked = model
		return m.stacked.Init()
	case intents.ExecJJ:
		if !m.revisions.InNormalMode() {
			return nil
		}
		return m.status.StartExec(common.ExecJJ)
	case intents.ExecShell:
		if !m.revisions.InNormalMode() {
			return nil
		}
		return m.status.StartExec(common.ExecShell)
	case intents.Quit:
		if !m.isSafeToQuit() {
			return nil
		}
		return tea.Quit
	case intents.Suspend:
		return tea.Suspend
	case intents.HelpToggle:
		return common.ToggleHelp
	case intents.OpenBookmarks:
		if !m.revisions.InNormalMode() {
			return nil
		}
		changeIds := m.revisions.GetCommitIds()
		model := bookmarks.NewModel(m.context, m.revisions.SelectedRevision(), changeIds)
		m.stacked = model
		return m.stacked.Init()
	case intents.OpenGit:
		if !m.revisions.InNormalMode() {
			return nil
		}
		model := git.NewModel(m.context, m.revisions.SelectedRevisions())
		m.stacked = model
		return m.stacked.Init()
	case intents.OpLogOpen:
		if !m.revisions.InNormalMode() {
			return nil
		}
		m.oplog = oplog.New(m.context)
		return m.oplog.Init()
	case intents.Edit:
		if !m.revisions.InNormalMode() {
			return nil
		}
		return m.revsetModel.Update(intent)
	case intents.PreviewToggle:
		m.previewModel.ToggleVisible()
		return common.SelectionChanged(m.context.SelectedItem)
	case intents.PreviewToggleBottom:
		previewPos := m.previewModel.AtBottom()
		m.previewModel.SetPosition(false, !previewPos)
		if m.previewModel.Visible() {
			return nil
		}
		m.previewModel.ToggleVisible()
		return common.SelectionChanged(m.context.SelectedItem)
	case intents.PreviewExpand:
		if !m.previewModel.Visible() {
			return nil
		}
		if m.revisionsSplit != nil && m.revisionsSplit.State != nil {
			m.revisionsSplit.State.Expand(config.Current.Preview.WidthIncrementPercentage)
		}
		return nil
	case intents.PreviewShrink:
		if !m.previewModel.Visible() {
			return nil
		}
		if m.revisionsSplit != nil && m.revisionsSplit.State != nil {
			m.revisionsSplit.State.Shrink(config.Current.Preview.WidthIncrementPercentage)
		}
		return nil
	case intents.PreviewScroll:
		if !m.previewModel.Visible() {
			return nil
		}
		switch intent.Kind {
		case intents.PreviewScrollUp:
			return m.previewModel.Scroll(-1)
		case intents.PreviewScrollDown:
			return m.previewModel.Scroll(1)
		case intents.PreviewHalfPageUp:
			return m.previewModel.HalfPageUp()
		case intents.PreviewHalfPageDown:
			return m.previewModel.HalfPageDown()
		}
		return nil
	case intents.QuickSearch:
		if m.oplog != nil {
			// HACK: prevents quick search from activating in op log view
			return nil
		}
		if !m.revisions.InNormalMode() {
			return nil
		}
		return m.status.StartQuickSearch()
	case intents.FileSearchToggle:
		rev := m.revisions.SelectedRevision()
		if rev == nil {
			// noop if current revset does not exist (#264)
			return nil
		}
		out, _ := m.context.RunCommandImmediate(jj.FilesInRevision(rev))
		return common.FileSearch(m.context.CurrentRevset, m.previewModel.Visible(), rev, out)
	case intents.OpenCustomCommands:
		model := customcommands.NewModel(m.context)
		m.stacked = model
		return m.stacked.Init()
	case intents.OpenLeader:
		m.leader = leader.New(m.context)
		return leader.InitCmd
	default:
		return nil
	}
}

func (m *Model) isSafeToQuit() bool {
	if m.stacked != nil {
		return false
	}
	if m.oplog != nil {
		return false
	}
	if m.revisions.InNormalMode() {
		return true
	}
	return false
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
	revisionsModel := revisions.New(c)
	statusModel := status.New(c)
	flashView := flash.New(c)
	previewModel := preview.New(c)
	revsetModel := revset.New(c)

	ui := &Model{
		context:      c,
		keyMap:       config.Current.GetKeyMap(),
		state:        common.Loading,
		revisions:    revisionsModel,
		previewModel: previewModel,
		status:       statusModel,
		revsetModel:  revsetModel,
		flash:        flashView,
	}
	ui.initSplit()
	return ui
}

func New(c *context.MainContext) tea.Model {
	return &wrapper{ui: NewUI(c)}
}
