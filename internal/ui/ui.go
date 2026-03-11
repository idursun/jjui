package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/idursun/jjui/internal/scripting"
	"github.com/idursun/jjui/internal/ui/actionmeta"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/helpkeys"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/password"
	"github.com/idursun/jjui/internal/ui/render"

	"github.com/idursun/jjui/internal/ui/commandhistory"
	"github.com/idursun/jjui/internal/ui/flash"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/bookmarkpane"
	"github.com/idursun/jjui/internal/ui/bookmarks"
	"github.com/idursun/jjui/internal/ui/choose"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/exec_process"
	"github.com/idursun/jjui/internal/ui/git"
	"github.com/idursun/jjui/internal/ui/help"

	"github.com/idursun/jjui/internal/ui/input"
	"github.com/idursun/jjui/internal/ui/oplog"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/redo"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/internal/ui/status"
	"github.com/idursun/jjui/internal/ui/undo"
)

type Model struct {
	revisions             *revisions.Model
	oplog                 *oplog.Model
	revsetModel           *revset.Model
	previewModel          *preview.Model
	diff                  *diff.Model
	flash                 *flash.Model
	state                 common.State
	status                *status.Model
	password              *password.Model
	context               *context.MainContext
	scriptRunner          *scripting.Runner
	configuredActions     map[keybindings.Action]config.ActionConfig
	paletteActions        map[string]keybindings.Action
	sequenceHelp          []helpkeys.Entry
	sequenceAutoOpen      bool
	resolver              *dispatch.Resolver
	stacked               common.StackedModel
	displayContext        *render.DisplayContext
	width                 int
	height                int
	activeSplit           *split
	splitActive           bool
	bookmarkPane          *bookmarkpane.Model
	secondaryPane         *secondaryPaneController
	bookmarkRevsetRestore string
}

type triggerAutoRefreshMsg struct{}

const (
	scopeUi               keybindings.Scope = "ui"
	scopePreview          keybindings.Scope = "ui.preview"
	scopeDiff             keybindings.Scope = "diff"
	scopeRevset           keybindings.Scope = "revset"
	scopeFileSearch       keybindings.Scope = "file_search"
	scopeQuickSearchInput keybindings.Scope = "revisions.quick_search.input"
	scopeOplogQuickSearch keybindings.Scope = "oplog.quick_search"
	scopePassword         keybindings.Scope = "password"
	scopeCommandHistory   keybindings.Scope = "command_history"
)

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.revisions.Init(), m.scheduleAutoRefresh())
}

func (m *Model) closeTopLayer(msg common.CloseViewMsg) (tea.Cmd, bool) {
	if m.diff != nil {
		m.diff = nil
		return nil, true
	}
	if m.stacked != nil {
		cmd := m.stacked.Update(msg)
		m.stacked = nil
		return cmd, true
	}
	if m.oplog != nil {
		m.oplog = nil
		return common.SelectionChanged(m.context.SelectedItem), true
	}
	return nil, false
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if closeMsg, ok := msg.(common.CloseViewMsg); ok {
		if cmd, handled := m.closeTopLayer(closeMsg); handled {
			return cmd
		}
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.ModeReportMsg:
		if msg.Mode == ansi.ModeUnicodeCore {
			if msg.Value == ansi.ModeReset || msg.Value == ansi.ModeSet || msg.Value == ansi.ModePermanentlySet {
				render.SetWidthMethod(ansi.GraphemeWidth)
			}
		}
		return nil
	case tea.FocusMsg:
		return common.RefreshAndKeepSelections
	case tea.MouseReleaseMsg:
		if m.splitActive {
			m.splitActive = false
		}
	case tea.MouseMotionMsg:
		if m.splitActive && m.activeSplit != nil {
			mouse := msg.Mouse()
			m.activeSplit.DragTo(mouse.X, mouse.Y)
			return nil
		}
	case tea.MouseClickMsg, tea.MouseWheelMsg:
		if m.displayContext != nil {
			if interactionMsg, handled := m.displayContext.ProcessMouseEvent(msg.(tea.MouseMsg)); handled {
				if interactionMsg != nil {
					return func() tea.Msg { return interactionMsg }
				}
				return nil
			}
		}
		return nil
	case tea.KeyMsg:
		if m.resolver != nil {
			result := m.resolver.ResolveKey(msg, m.dispatchScopes())
			if result.Pending {
				m.setSequenceStatusHelp(result.Continuations)
				return nil
			}
			m.clearSequenceStatusHelp()
			if result.LuaScript != "" {
				return luaCmd(result.LuaScript)
			}
			if result.Intent != nil {
				return m.routeIntent(result.Owner, result.Intent)
			}
			if result.Consumed {
				return nil
			}
		}
		return m.handleUnmatched(msg)
	case intents.Intent:
		if cmd := m.handleIntent(msg); cmd != nil {
			return cmd
		}
	case common.ExecMsg:
		return exec_process.ExecLine(m.context, msg)
	case common.ExecProcessCompletedMsg:
		cmds = append(cmds, common.Refresh)
	case common.UpdateRevisionsSuccessMsg:
		m.state = common.Ready
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
		m.revsetModel.Update(msg)
		return common.Refresh
	case common.RunLuaScriptMsg:
		if m.scriptRunner != nil && !m.scriptRunner.Done() {
			err := fmt.Errorf("lua script is already running")
			return intents.Invoke(intents.AddMessage{Text: err.Error(), Err: err})
		}
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
	case common.DispatchActionMsg:
		if actionmeta.IsBuiltInAction(msg.Action) {
			if err := actionmeta.ValidateBuiltInActionArgs(msg.Action, msg.Args); err != nil {
				return intents.Invoke(intents.AddMessage{Text: err.Error(), Err: err})
			}
		}
		action := keybindings.Action(strings.TrimSpace(msg.Action))
		var result dispatch.Result
		if msg.BuiltIn {
			result = m.resolver.ResolveBuiltInAction(action, msg.Args)
		} else {
			result = m.resolver.ResolveAction(action, msg.Args)
		}
		if result.LuaScript != "" {
			return luaCmd(result.LuaScript)
		}
		if result.Intent != nil {
			return m.routeIntent(result.Owner, result.Intent)
		}
		return nil
	case common.ShowChooseMsg:
		model := choose.NewWithOptions(msg.Options, msg.Title, msg.Filter, msg.Ordered)
		m.stacked = model
		return m.stacked.Init()
	case choose.SelectedMsg:
		m.stacked = nil
		if action, ok := m.paletteActions[msg.Value]; ok {
			m.paletteActions = nil
			result := m.resolver.ResolveAction(action, nil)
			if result.LuaScript != "" {
				return luaCmd(result.LuaScript)
			}
			if result.Intent != nil {
				return m.routeIntent(result.Owner, result.Intent)
			}
			return nil
		}
		m.paletteActions = nil
	case choose.CancelledMsg:
		m.stacked = nil
		m.paletteActions = nil
	case common.ShowInputMsg:
		model := input.NewWithTitle(msg.Title, msg.Prompt, msg.InitialValue)
		m.stacked = model
		return m.stacked.Init()
	case input.SelectedMsg, input.CancelledMsg:
		m.stacked = nil
	case common.ShowPreview:
		if m.secondaryPane != nil {
			if bool(msg) {
				if m.secondaryPane.bookmarkVisible() {
					cmds = append(cmds, m.closeBookmarkPane())
				}
				m.secondaryPane.showPreview()
			} else {
				m.secondaryPane.hidePreview()
			}
		}
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
	if m.bookmarkPane != nil {
		cmds = append(cmds, m.bookmarkPane.Update(msg))
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
	mode := m.statusMode()
	entries := m.bindingStatusHelp()

	if m.sequenceHelp != nil {
		entries = m.sequenceHelp
	}

	if mode != "" {
		m.status.SetMode(mode)
	}
	m.status.SetHelp(entries)
}

func (m *Model) statusMode() string {
	switch {
	case m.commandHistoryOpen():
		return "history"
	case m.secondaryPane != nil && m.secondaryPane.bookmarkVisible() && m.secondaryPane.bookmarkFocused:
		return actions.OwnerBookmarkView
	case m.stacked != nil:
		return m.stacked.StackedActionOwner()
	case m.diff != nil:
		return "diff"
	case m.oplog != nil:
		return "oplog"
	case m.revsetModel.Editing:
		return "revset"
	default:
		return m.revisions.CurrentOperation().Name()
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

	box := layout.NewBox(layout.Rect(0, 0, m.width, m.height))
	screenBuf := render.NewScreenBuffer(m.width, m.height)

	if m.diff != nil {
		m.renderDiffLayout(box)
	} else {
		if m.secondaryPane != nil && m.secondaryPane.previewVisible() {
			m.UpdatePreviewPosition()
		}
		if m.secondaryPane != nil {
			m.secondaryPane.syncPreviewOrientation()
		}
		if m.oplog != nil {
			m.renderOpLogLayout(box)
		} else {
			m.renderRevisionsLayout(box)
		}
	}

	if m.stacked != nil {
		m.stacked.ViewRect(m.displayContext, box)
	}

	if !m.commandHistoryOpen() {
		m.flash.ViewRect(m.displayContext, box)
	}

	if m.password != nil {
		m.password.ViewRect(m.displayContext, box)
	}

	m.displayContext.Render(screenBuf)
	finalView := screenBuf.Render()
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
	if m.secondaryPane == nil {
		primary.ViewRect(m.displayContext, box)
		return
	}
	m.secondaryPane.render(primary, m.displayContext, box)
}

func (m *Model) initSplit() {
	m.secondaryPane = newSecondaryPaneController(m.previewModel, m.bookmarkPane)
}

func (m *Model) openBookmarkPane() tea.Cmd {
	if m.secondaryPane == nil {
		return nil
	}
	return m.secondaryPane.openBookmark()
}

func (m *Model) closeBookmarkPane() tea.Cmd {
	if m.secondaryPane == nil {
		return nil
	}
	m.secondaryPane.closeBookmark()
	if m.bookmarkRevsetRestore != "" {
		restore := m.bookmarkRevsetRestore
		m.bookmarkRevsetRestore = ""
		return common.UpdateRevSet(restore)
	}
	return nil
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

// handleIntent is the UI-root intent boundary.
// - Delegated intents are owned by child models and forwarded.
// - UI-root intents mutate top-level UI state, view composition, or lifecycle.
func (m *Model) handleIntent(intent intents.Intent) tea.Cmd {
	if cmd, handled := m.handleDelegatedIntent(intent); handled {
		return cmd
	}
	if cmd, handled := m.handleUiRootIntent(intent); handled {
		return cmd
	}
	return nil
}

func (m *Model) handleDelegatedIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Edit:
		if !m.revisions.InNormalMode() {
			return nil, true
		}
		return m.revsetModel.Update(intent), true
	case intents.DiffShow:
		if m.diff == nil {
			m.diff = diff.New("")
		}
		return m.diff.Update(intent), true
	case intents.PreviewShow:
		if !m.previewModel.Visible() {
			m.previewModel.ToggleVisible()
		}
		return m.previewModel.Update(intent), true
	default:
		return nil, false
	}
}

func (m *Model) handleUiRootIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Cancel:
		return m.routeCancel("", intent), true
	case intents.Undo:
		if !m.revisions.InNormalMode() {
			return nil, true
		}
		model := undo.NewModel(m.context)
		m.stacked = model
		return m.stacked.Init(), true
	case intents.Redo:
		if !m.revisions.InNormalMode() {
			return nil, true
		}
		model := redo.NewModel(m.context)
		m.stacked = model
		return m.stacked.Init(), true
	case intents.ExecJJ:
		if !m.revisions.InNormalMode() {
			return nil, true
		}
		return m.status.StartExec(common.ExecJJ), true
	case intents.ExecShell:
		if !m.revisions.InNormalMode() {
			return nil, true
		}
		return m.status.StartExec(common.ExecShell), true
	case intents.Quit:
		return tea.Quit, true
	case intents.Suspend:
		return tea.Suspend, true
	case intents.ExpandStatusToggle:
		m.status.ToggleStatusExpand()
		return nil, true
	case intents.OpenHelp:
		if m.stacked != nil || m.diff != nil {
			return nil, true
		}
		model := help.New()
		m.stacked = model
		return m.stacked.Init(), true
	case intents.OpenBookmarks:
		if !m.revisions.InNormalMode() {
			return nil, true
		}
		current := m.revisions.SelectedRevision()
		if current == nil {
			return nil, true
		}
		changeIds := m.revisions.GetCommitIds()
		model := bookmarks.NewModel(m.context, current, changeIds)
		m.stacked = model
		return m.stacked.Init(), true
	case intents.OpenGit:
		if !m.revisions.InNormalMode() {
			return nil, true
		}
		model := git.NewModel(m.context, m.revisions.SelectedRevisions())
		m.stacked = model
		return m.stacked.Init(), true
	case intents.OpLogOpen:
		if !m.revisions.InNormalMode() {
			return nil, true
		}
		m.oplog = oplog.New(m.context)
		return m.oplog.Init(), true
	case intents.PreviewToggle:
		var closeCmd tea.Cmd
		restorePreview := m.secondaryPane != nil && m.secondaryPane.restoreOnClose == secondaryPanePreview
		if m.secondaryPane != nil && m.secondaryPane.bookmarkVisible() {
			closeCmd = m.closeBookmarkPane()
			if restorePreview {
				return tea.Batch(closeCmd, common.SelectionChanged(m.context.SelectedItem)), true
			}
		}
		if m.secondaryPane != nil {
			m.secondaryPane.togglePreview()
		}
		return tea.Batch(closeCmd, common.SelectionChanged(m.context.SelectedItem)), true
	case intents.PreviewToggleBottom:
		var closeCmd tea.Cmd
		restorePreview := m.secondaryPane != nil && m.secondaryPane.restoreOnClose == secondaryPanePreview
		if m.secondaryPane != nil && m.secondaryPane.bookmarkVisible() {
			closeCmd = m.closeBookmarkPane()
			if restorePreview {
				previewPos := m.previewModel.AtBottom()
				m.previewModel.SetPosition(false, !previewPos)
				return closeCmd, true
			}
		}
		previewPos := m.previewModel.AtBottom()
		m.previewModel.SetPosition(false, !previewPos)
		if m.secondaryPane != nil && m.secondaryPane.previewVisible() {
			return closeCmd, true
		}
		if m.secondaryPane != nil {
			m.secondaryPane.showPreview()
		}
		return tea.Batch(closeCmd, common.SelectionChanged(m.context.SelectedItem)), true
	case intents.PreviewExpand:
		if m.secondaryPane == nil || !m.secondaryPane.previewVisible() {
			return nil, true
		}
		if split := m.secondaryPane.previewSplit; split != nil && split.State != nil {
			split.State.Expand(config.Current.Preview.WidthIncrementPercentage)
		}
		return nil, true
	case intents.PreviewShrink:
		if m.secondaryPane == nil || !m.secondaryPane.previewVisible() {
			return nil, true
		}
		if split := m.secondaryPane.previewSplit; split != nil && split.State != nil {
			split.State.Shrink(config.Current.Preview.WidthIncrementPercentage)
		}
		return nil, true
	case intents.PreviewScroll:
		if m.secondaryPane == nil || !m.secondaryPane.previewVisible() {
			return nil, true
		}
		switch intent.Kind {
		case intents.PreviewScrollUp:
			return m.previewModel.Scroll(-1), true
		case intents.PreviewScrollDown:
			return m.previewModel.Scroll(1), true
		case intents.PreviewPageUp:
			return m.previewModel.PageUp(), true
		case intents.PreviewPageDown:
			return m.previewModel.PageDown(), true
		case intents.PreviewHalfPageUp:
			return m.previewModel.HalfPageUp(), true
		case intents.PreviewHalfPageDown:
			return m.previewModel.HalfPageDown(), true
		}
		return nil, true
	case intents.QuickSearch:
		if m.oplog == nil && !m.revisions.InNormalMode() {
			return nil, true
		}
		return m.status.StartQuickSearch(), true
	case intents.FileSearchToggle:
		rev := m.revisions.SelectedRevision()
		if rev == nil {
			// noop if current revset does not exist (#264)
			return nil, true
		}
		out, _ := m.context.RunCommandImmediate(jj.FilesInRevision(rev))
		return common.FileSearch(m.context.CurrentRevset, m.secondaryPane != nil && m.secondaryPane.previewVisible(), rev, out), true
	case intents.CommandHistoryToggle:
		if m.commandHistoryOpen() {
			m.stacked = nil
			return nil, true
		}
		m.stacked = commandhistory.New(m.context, m.flash)
		return m.stacked.Init(), true
	case intents.ToggleBookmarkView:
		if !m.revisions.InNormalMode() {
			return nil, true
		}
		if m.secondaryPane != nil && m.secondaryPane.bookmarkVisible() {
			return m.closeBookmarkPane(), true
		}
		return m.openBookmarkPane(), true
	case intents.FocusNextPane:
		if m.secondaryPane == nil || !m.secondaryPane.bookmarkVisible() {
			return nil, true
		}
		m.secondaryPane.focusNext()
		return nil, true
	default:
		return nil, false
	}
}

func luaCmd(script string) tea.Cmd {
	return func() tea.Msg {
		return common.RunLuaScriptMsg{Script: script}
	}
}

func (m *Model) routeIntent(owner string, intent intents.Intent) tea.Cmd {
	// Cancel has priority-based routing that depends on UI state.
	if cancel, ok := intent.(intents.Cancel); ok {
		return m.routeCancel(owner, cancel)
	}

	if cmd, handled := m.routeIntentByOwner(owner, intent); handled {
		return cmd
	}
	return m.handleIntent(intent)
}

func (m *Model) routeCancel(owner string, cancel intents.Cancel) tea.Cmd {
	if m.secondaryPane != nil && m.secondaryPane.bookmarkVisible() && m.flash.Any() {
		m.flash.DeleteOldest()
		return nil
	}

	if m.stacked != nil || m.diff != nil || m.oplog != nil {
		return common.Close
	}

	if m.clearCancelUIState() {
		return nil
	}

	if cmd, handled := m.routeIntentByOwner(owner, cancel); handled {
		return cmd
	}

	if m.shouldRouteCancelToRevisions() {
		if cmd, handled := m.revisions.HandleDispatchedAction(actions.UiCancel, nil); handled {
			return cmd
		}
	}
	return nil
}

func (m *Model) clearCancelUIState() bool {
	switch {
	case m.flash.Any():
		m.flash.DeleteOldest()
		return true
	case m.status.StatusExpanded():
		m.status.ToggleStatusExpand()
		return true
	default:
		return false
	}
}

func (m *Model) routeIntentByOwner(owner string, intent intents.Intent) (tea.Cmd, bool) {
	switch owner {
	case actions.OwnerPassword:
		if m.password != nil {
			return m.password.Update(intent), true
		}
	case actions.OwnerStatusInput, actions.OwnerFileSearch, actions.OwnerQuickSearchInput:
		if m.status.IsFocused() {
			return m.status.Update(intent), true
		}
	case actions.OwnerRevset:
		return m.revsetModel.Update(intent), true
	case actions.OwnerDiff:
		if m.diff != nil {
			return m.diff.Update(intent), true
		}
	case actions.OwnerUiPreview:
		if m.previewModel.Visible() {
			return m.previewModel.Update(intent), true
		}
	case actions.OwnerOplog, actions.OwnerOplogQuickSearch:
		if m.oplog != nil {
			return m.oplog.Update(intent), true
		}
	case actions.OwnerCommandHistory,
		actions.OwnerBookmarks,
		actions.OwnerBookmarkView,
		actions.OwnerGit,
		actions.OwnerChoose,
		actions.OwnerUndo,
		actions.OwnerRedo,
		actions.OwnerInput,
		actions.OwnerHelp:
		if m.stacked != nil && owner == m.stacked.StackedActionOwner() {
			return m.stacked.Update(intent), true
		}
		if m.bookmarkPane != nil && owner == actions.OwnerBookmarkView {
			return m.bookmarkPane.Update(intent), true
		}
	default:
		if cmd, handled := m.revisions.RouteOwnedIntent(owner, intent); handled {
			return cmd, true
		}
	}

	return nil, false
}

func (m *Model) shouldRouteCancelToRevisions() bool {
	if m.status.IsFocused() {
		return false
	}
	if m.revsetModel.Editing || m.revisions.IsEditing() {
		return false
	}
	if m.revisions.HasQuickSearch() {
		return false
	}
	if m.flash.Any() || m.status.StatusExpanded() {
		return false
	}
	return m.revisions.InNormalMode()
}

func (m *Model) handleUnmatched(msg tea.KeyMsg) tea.Cmd {
	if m.commandHistoryOpen() {
		return nil
	}

	if m.status.IsFocused() {
		return m.status.Update(msg)
	}

	if m.revsetModel.Editing {
		m.state = common.Loading
		return m.revsetModel.Update(msg)
	}

	if m.stacked != nil {
		return m.stacked.Update(msg)
	}
	if m.secondaryPane != nil && m.secondaryPane.bookmarkVisible() && m.secondaryPane.bookmarkFocused {
		return m.bookmarkPane.Update(msg)
	}
	if m.diff != nil {
		return m.diff.Update(msg)
	}
	if m.oplog != nil {
		return nil
	}
	return m.revisions.Update(msg)
}

func (m *Model) primaryScope() keybindings.Scope {
	if m.password != nil {
		return scopePassword
	}

	switch m.status.FocusKind() {
	case status.FocusFileSearch:
		return scopeFileSearch
	case status.FocusInput:
		return actions.OwnerStatusInput
	case status.FocusQuickSearch:
		return scopeQuickSearchInput
	default:
	}

	if m.revsetModel.Editing {
		return scopeRevset
	}

	if m.diff != nil {
		return scopeDiff
	}

	if m.stacked != nil {
		if e, ok := m.stacked.(common.Editable); ok && e.IsEditing() {
			return keybindings.Scope(m.stacked.StackedActionOwner() + ".filter")
		}
		return keybindings.Scope(m.stacked.StackedActionOwner())
	}

	if m.secondaryPane != nil {
		if scope := m.secondaryPane.primaryScope(); scope != "" {
			return scope
		}
	}

	if m.revisions.HasQuickSearch() {
		return revisions.ScopeQuickSearch
	}

	if m.oplog != nil {
		return actions.OwnerOplog
	}

	scopes := m.revisions.ScopeChain()
	if len(scopes) == 0 {
		return revisions.ScopeRevisions
	}
	return scopes[0]
}

func (m *Model) alwaysOnScopes() []keybindings.Scope {
	if m.status.IsFocused() || m.revsetModel.Editing || m.revisions.IsEditing() {
		return nil
	}
	if m.stacked != nil {
		if f, ok := m.stacked.(common.Focusable); ok && f.IsFocused() {
			return nil
		}
	}
	if m.secondaryPane != nil && m.secondaryPane.bookmarkEditing() {
		return nil
	}
	scopes := []keybindings.Scope{scopeUi}
	if m.secondaryPane != nil && m.secondaryPane.previewVisible() {
		scopes = append(scopes, scopePreview)
	}
	return scopes
}

func (m *Model) dispatchScopes() []keybindings.Scope {
	if m.commandHistoryOpen() {
		return []keybindings.Scope{scopeCommandHistory}
	}
	primary := m.primaryScope()
	if primary == "" {
		return nil
	}
	var scopes []keybindings.Scope
	if m.oplog != nil && m.oplog.HasQuickSearch() {
		scopes = append(scopes, scopeOplogQuickSearch)
	}
	scopes = append(scopes, primary)
	for _, scope := range m.alwaysOnScopes() {
		if scope != "" && scope != primary {
			scopes = append(scopes, scope)
		}
	}
	return scopes
}

func (m *Model) commandHistoryOpen() bool {
	return m.stacked != nil && m.stacked.StackedActionOwner() == actions.OwnerCommandHistory
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

func (w *wrapper) View() tea.View {
	if w.render {
		w.cachedFrame = w.ui.View()
		w.render = false
	}
	v := tea.NewView(w.cachedFrame)
	v.WindowTitle = fmt.Sprintf("jjui - %s", w.ui.context.Location)
	v.AltScreen = true
	v.ReportFocus = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m *Model) showBookmarkTarget(target, commitID string) tea.Cmd {
	if target == "" {
		return nil
	}
	if m.bookmarkRevsetRestore == "" {
		m.bookmarkRevsetRestore = m.context.CurrentRevset
	}
	return common.UpdateRevSet(fmt.Sprintf("::%s", target))
}

func NewUI(c *context.MainContext) *Model {
	revisionsModel := revisions.New(c)
	statusModel := status.New(c)
	flashView := flash.New(c)
	previewModel := preview.New(c)
	revsetModel := revset.New(c)
	var ui *Model
	bookmarkPaneModel := bookmarkpane.NewModel(c, bookmarkpane.Callbacks{
		CurrentRevision: revisionsModel.SelectedRevision,
		RevealVisible:   revisionsModel.RevealRevision,
		ShowInRevisions: func(target, commitID string) tea.Cmd {
			if ui == nil {
				return nil
			}
			return ui.showBookmarkTarget(target, commitID)
		},
	})

	ui = &Model{
		context:           c,
		state:             common.Loading,
		revisions:         revisionsModel,
		previewModel:      previewModel,
		status:            statusModel,
		revsetModel:       revsetModel,
		flash:             flashView,
		bookmarkPane:      bookmarkPaneModel,
		configuredActions: make(map[keybindings.Action]config.ActionConfig),
	}
	ui.initConfiguredActions()
	ui.initResolver()
	ui.initSplit()
	return ui
}

func (m *Model) initConfiguredActions() {
	for _, action := range config.Current.Actions {
		name := keybindings.Action(strings.TrimSpace(action.Name))
		if name == "" {
			continue
		}
		m.configuredActions[name] = action
	}
}

func (m *Model) bindingStatusHelp() []helpkeys.Entry {
	scopes := m.dispatchScopes()
	if len(scopes) == 0 {
		return nil
	}
	return helpkeys.BuildFromBindings(scopes, config.Current.Bindings)
}

func (m *Model) setSequenceStatusHelp(continuations []dispatch.Continuation) {
	entries := helpkeys.BuildFromContinuations(continuations)
	if len(entries) == 0 {
		return
	}

	if m.sequenceHelp == nil {
		if !m.status.StatusExpanded() {
			m.status.SetStatusExpanded(true)
			m.sequenceAutoOpen = true
		} else {
			m.sequenceAutoOpen = false
		}
	}
	m.sequenceHelp = entries
}

func (m *Model) clearSequenceStatusHelp() {
	if m.sequenceHelp == nil {
		return
	}
	m.sequenceHelp = nil
	if m.sequenceAutoOpen {
		m.status.SetStatusExpanded(false)
	}
	m.sequenceAutoOpen = false
}

func (m *Model) initResolver() {
	bindings := config.BindingsToRuntime(config.Current.Bindings)
	dispatcher, err := dispatch.NewDispatcher(bindings)
	if err != nil {
		return
	}
	m.resolver = dispatch.NewResolver(dispatcher, m.configuredActions)
}

func New(c *context.MainContext) tea.Model {
	return &wrapper{ui: NewUI(c)}
}
