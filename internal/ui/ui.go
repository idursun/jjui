package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/scripting"
	"github.com/idursun/jjui/internal/ui/actionmeta"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/flash"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	bookmarkop "github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/password"
	"github.com/idursun/jjui/internal/ui/render"

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
	revisions               *revisions.Model
	oplog                   *oplog.Model
	revsetModel             *revset.Model
	previewModel            *preview.Model
	diff                    *diff.Model
	flash                   *flash.Model
	state                   common.State
	status                  *status.Model
	password                *password.Model
	context                 *context.MainContext
	scriptRunner            *scripting.Runner
	sequenceHelp            []help.Entry
	sequenceAutoOpen        bool
	resolver                *dispatch.Resolver
	stacked                 common.StackedModel
	displayContext          *render.DisplayContext
	width                   int
	height                  int
	activeSplit             *split
	previewSplit            *split
	bookmarkSplit           *split
	secondaryPaneActive     secondaryPaneKind
	secondaryRestoreOnClose secondaryPaneKind
	bookmarkPaneFocused     bool
	bookmarkPane            *bookmarkpane.Model
	bookmarkRevsetRestore   string
	bookmarkRevsetApplied   string
}

type triggerAutoRefreshMsg struct{}

type secondaryPaneKind int

const (
	secondaryPaneNone secondaryPaneKind = iota
	secondaryPanePreview
	secondaryPaneBookmark
)

const (
	scopeUi keybindings.ScopeName = "ui"
)

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.revisions.Init(), m.scheduleAutoRefresh())
}

func (m *Model) closeTopScope(msg common.CloseViewMsg) (tea.Cmd, bool) {
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
	if m.bookmarkVisible() && m.bookmarkPaneFocused {
		return m.closeBookmarkPane(), true
	}
	return nil, false
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if closeMsg, ok := msg.(common.CloseViewMsg); ok {
		if cmd, handled := m.closeTopScope(closeMsg); handled {
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
		if m.state == common.Ready {
			return common.RefreshAndKeepSelections
		}
		return nil
	case tea.MouseReleaseMsg:
		m.activeSplit = nil
	case tea.MouseMotionMsg:
		if m.activeSplit != nil {
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
			scopes := m.dispatchScopes()
			result := m.resolver.ResolveKey(msg, scopes)
			if result.Pending {
				m.setSequenceStatusHelp(result.Continuations)
				return nil
			}
			m.clearSequenceStatusHelp()
			if result.LuaScript != "" {
				return luaCmd(result.LuaScript)
			}
			if result.Intent != nil {
				start := slices.IndexFunc(scopes, func(scope dispatch.Scope) bool {
					return string(scope.Name) == result.Scope
				})
				if start < 0 {
					return nil
				}
				if cmd, handled := dispatch.RouteIntent(scopes[start:], result.Intent); handled {
					return cmd
				}
				if scopes[start].Leak != dispatch.LeakAll {
					return m.updateBlockingScope(scopes[start], msg)
				}
				return nil
			}
			if result.Consumed {
				return nil
			}

			for _, scope := range scopes {
				if scope.Leak != dispatch.LeakAll {
					return m.updateBlockingScope(scope, msg)
				}
			}
			return nil
		}
		return nil
	case intents.Intent:
		if cmd, handled := m.HandleIntent(msg); handled {
			return cmd
		}
	case common.ExecMsg:
		return exec_process.ExecLine(m.context, msg)
	case common.ExecProcessCompletedMsg:
		cmds = append(cmds, common.Refresh)
	case common.UpdateRevisionsSuccessMsg:
		m.state = common.Ready
		m.syncBookmarkPaneContext()
	case common.SelectionChangedMsg:
		m.syncBookmarkPaneContext()
	case common.FocusBookmarkViewMsg:
		if m.bookmarkVisible() {
			m.focusBookmarkPane()
		}
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
			if result.Scope == actions.ScopeRevset {
				return m.revsetModel.Update(result.Intent)
			}
			scopes := m.dispatchScopes()
			cmd, _ := dispatch.RouteIntent(scopes, result.Intent)
			return cmd
		}
		return nil
	case common.ShowChooseMsg:
		model := choose.NewWithOptions(msg.Options, msg.Title, msg.Filter, msg.Ordered)
		m.stacked = model
		return m.stacked.Init()
	case choose.SelectedMsg:
		m.stacked = nil
	case choose.CancelledMsg:
		m.stacked = nil
	case common.ShowInputMsg:
		model := input.NewWithTitle(msg.Title, msg.Prompt, msg.InitialValue)
		m.stacked = model
		return m.stacked.Init()
	case input.SelectedMsg, input.CancelledMsg:
		m.stacked = nil
	case bookmarkpane.RevealBookmarkMsg:
		cmd := m.revisions.RevealRevision(msg.CommitID)
		if m.bookmarkVisible() && m.bookmarkPaneFocused {
			m.focusNextPane()
		}
		return cmd
	case bookmarkpane.ShowBookmarkInRevisionsMsg:
		return m.showBookmarkTarget(msg.Target, msg.CommitID)
	case bookmarkpane.BeginMoveBookmarkMsg:
		op := bookmarkop.NewMoveBookmarkOperation(m.context, msg.Name)
		cmds = append(cmds, common.RestoreOperation(op))
		if m.bookmarkVisible() && m.bookmarkPaneFocused {
			m.focusNextPane()
		}
		return tea.Batch(cmds...)
	case common.ShowPreview:
		if bool(msg) {
			if m.bookmarkVisible() {
				cmds = append(cmds, m.closeBookmarkPane())
			}
			m.showPreview()
		} else {
			m.hidePreview()
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
	if mode != "" {
		m.status.SetMode(mode)
	}

	if m.sequenceHelp != nil {
		m.status.SetHelp(m.sequenceHelp)
	} else {
		m.status.SetScopes(m.dispatchScopes())
	}
}

func (m *Model) statusMode() string {
	if scope, ok := m.stackedScope(); ok {
		if scope == actions.ScopeCommandHistory {
			return "history"
		}
		return string(scope)
	}

	switch {
	case m.bookmarkVisible() && m.bookmarkPaneFocused:
		if m.bookmarkEditing() {
			return actions.ScopeBookmarkViewFilter
		}
		return actions.ScopeBookmarkView
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
		if m.previewVisible() {
			m.UpdatePreviewPosition()
		}
		m.syncPreviewSplitOrientation()
		if m.oplog != nil {
			m.renderOpLogLayout(box)
		} else {
			m.renderRevisionsLayout(box)
		}
	}

	if m.stacked != nil {
		m.stacked.ViewRect(m.displayContext, box)
	}

	if scope, ok := m.stackedScope(); !ok || scope != actions.ScopeCommandHistory {
		flashBox, _ := box.CutBottom(1)
		m.flash.ViewRect(m.displayContext, flashBox)
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
	switch m.secondaryPaneActive {
	case secondaryPaneBookmark:
		if m.bookmarkSplit == nil {
			return
		}
		m.bookmarkSplit.Primary = primary
		m.bookmarkSplit.Secondary = m.bookmarkPane
		m.bookmarkSplit.Vertical = false
		m.bookmarkSplit.SeparatorVisible = true
		m.bookmarkSplit.Render(m.displayContext, box)
	case secondaryPanePreview:
		if m.previewSplit == nil {
			return
		}
		m.previewSplit.Primary = primary
		m.previewSplit.Secondary = m.previewModel
		m.previewSplit.Render(m.displayContext, box)
	default:
		primary.ViewRect(m.displayContext, box)
	}
}

func (m *Model) initSplit() {
	m.previewSplit = newSplit(
		newSplitState(config.Current.Preview.WidthPercentage),
		nil,
		m.previewModel,
	)
	m.bookmarkSplit = newSplit(
		newSplitState(45),
		nil,
		m.bookmarkPane,
	)
}

func (m *Model) previewVisible() bool {
	return m.secondaryPaneActive == secondaryPanePreview && m.previewModel != nil && m.previewModel.Visible()
}

func (m *Model) bookmarkVisible() bool {
	return m.secondaryPaneActive == secondaryPaneBookmark && m.bookmarkPane != nil && m.bookmarkPane.Visible()
}

func (m *Model) bookmarkEditing() bool {
	return m.bookmarkVisible() && m.bookmarkPane != nil && m.bookmarkPane.IsEditing()
}

func (m *Model) syncPreviewSplitOrientation() {
	if m.previewSplit == nil || m.previewModel == nil {
		return
	}
	m.previewSplit.Vertical = m.previewModel.AtBottom()
}

func (m *Model) setBookmarkPaneFocused(focused bool) {
	m.bookmarkPaneFocused = focused
	if m.bookmarkPane != nil {
		m.bookmarkPane.SetFocused(focused)
	}
	m.revisions.SetFocused(!focused)
}

func (m *Model) focusBookmarkPane() {
	if !m.bookmarkVisible() {
		return
	}
	m.setBookmarkPaneFocused(true)
}

func (m *Model) focusNextPane() {
	if !m.bookmarkVisible() {
		return
	}
	m.setBookmarkPaneFocused(!m.bookmarkPaneFocused)
}

func (m *Model) showPreview() {
	if m.previewModel == nil {
		return
	}
	m.previewModel.SetVisible(true)
	m.secondaryPaneActive = secondaryPanePreview
}

func (m *Model) hidePreview() {
	if m.previewModel == nil {
		return
	}
	m.previewModel.SetVisible(false)
	if m.secondaryPaneActive == secondaryPanePreview {
		m.secondaryPaneActive = secondaryPaneNone
	}
}

func (m *Model) togglePreview() {
	if m.previewVisible() {
		m.hidePreview()
		return
	}
	m.showPreview()
}

func (m *Model) syncBookmarkPaneContext() {
	if m.bookmarkPane == nil {
		return
	}
	if selected := m.revisions.SelectedRevision(); selected != nil {
		m.bookmarkPane.SetCurrentCommitID(selected.CommitId)
	} else {
		m.bookmarkPane.SetCurrentCommitID("")
	}
	m.bookmarkPane.SetVisibleCommitIDs(m.revisions.GetCommitIds())
}

func (m *Model) openBookmarkPane() tea.Cmd {
	m.syncBookmarkPaneContext()
	m.secondaryRestoreOnClose = secondaryPaneNone
	if m.previewVisible() {
		m.secondaryRestoreOnClose = secondaryPanePreview
		m.previewModel.SetVisible(false)
	}
	m.secondaryPaneActive = secondaryPaneBookmark
	m.setBookmarkPaneFocused(true)
	return m.bookmarkPane.Open()
}

func (m *Model) closeBookmarkPane() tea.Cmd {
	m.bookmarkPane.Close()
	m.bookmarkPaneFocused = false
	m.secondaryPaneActive = secondaryPaneNone
	m.revisions.SetFocused(true)
	if m.secondaryRestoreOnClose == secondaryPanePreview {
		m.previewModel.SetVisible(true)
		m.secondaryPaneActive = secondaryPanePreview
	}
	m.secondaryRestoreOnClose = secondaryPaneNone
	if m.bookmarkRevsetRestore != "" && m.context.CurrentRevset == m.bookmarkRevsetApplied {
		restore := m.bookmarkRevsetRestore
		m.bookmarkRevsetRestore = ""
		m.bookmarkRevsetApplied = ""
		return common.UpdateRevSet(restore)
	}
	m.bookmarkRevsetRestore = ""
	m.bookmarkRevsetApplied = ""
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

func (m *Model) dispatchScopes() []dispatch.Scope {
	var scopes []dispatch.Scope

	if m.password != nil {
		scopes = append(scopes, m.password.Scopes()...)
	}
	scopes = append(scopes, m.status.Scopes()...)
	scopes = append(scopes, m.revsetModel.Scopes()...)

	if m.diff != nil {
		scopes = append(scopes, m.diff.Scopes()...)
	}
	if m.stacked != nil {
		scopes = append(scopes, m.stacked.Scopes()...)
	} else if m.bookmarkVisible() && m.bookmarkPaneFocused {
		scopes = append(scopes, m.bookmarkPaneScopes()...)
	} else if m.oplog != nil {
		scopes = append(scopes, m.oplog.Scopes()...)
	} else {
		scopes = append(scopes, m.revisions.Scopes()...)
	}

	scopes = append(scopes, m.previewModel.Scopes()...)
	scopes = append(scopes, dispatch.Scope{
		Name:    scopeUi,
		Leak:    dispatch.LeakNone,
		Global:  true,
		Handler: m,
	})

	return scopes
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	if cmd, handled := m.handleBookmarkPaneIntent(intent); handled {
		return cmd, true
	}

	switch intent := intent.(type) {

	// --- Quit / Suspend ---
	case intents.Quit:
		return tea.Quit, true
	case intents.Suspend:
		return tea.Suspend, true

	// --- Cancel fallback (only reached if no inner scope handled it) ---
	case intents.Cancel:
		if m.stacked != nil || m.diff != nil || m.oplog != nil {
			return common.Close, true
		}
		if m.flash.Any() {
			m.flash.DeleteOldest()
			return nil, true
		}
		if m.status.StatusExpanded() {
			m.status.ToggleStatusExpand()
			return nil, true
		}
		return nil, false

	// --- Open stacked views ---
	case intents.OpenGit:
		model := git.NewModel(m.context, m.revisions.SelectedRevisions())
		m.stacked = model
		return m.stacked.Init(), true
	case intents.OpenBookmarks:
		current := m.revisions.SelectedRevision()
		if current == nil {
			return nil, true
		}
		changeIds := m.revisions.GetCommitIds()
		model := bookmarks.NewModel(m.context, current, changeIds)
		m.stacked = model
		return m.stacked.Init(), true
	case intents.OpLogOpen:
		m.oplog = oplog.New(m.context)
		return m.oplog.Init(), true
	case intents.Undo:
		model := undo.NewModel(m.context)
		m.stacked = model
		return m.stacked.Init(), true
	case intents.Redo:
		model := redo.NewModel(m.context)
		m.stacked = model
		return m.stacked.Init(), true
	case intents.OpenHelp:
		if m.stacked != nil || m.diff != nil {
			return nil, true
		}
		model := help.New()
		m.stacked = model
		return m.stacked.Init(), true
	case intents.CommandHistoryToggle:
		if scope, ok := m.stackedScope(); ok && scope == actions.ScopeCommandHistory {
			m.stacked = nil
			return nil, true
		}
		m.stacked = m.flash.NewHistory()
		return m.stacked.Init(), true

	// --- Activate input modes ---
	case intents.Edit:
		return m.revsetModel.Update(intent), true
	case intents.ExecJJ:
		return m.status.StartExec(common.ExecJJ), true
	case intents.ExecShell:
		return m.status.StartExec(common.ExecShell), true
	case intents.QuickSearch:
		return m.status.StartQuickSearch(), true
	case intents.FileSearchToggle:
		rev := m.revisions.SelectedRevision()
		if rev == nil {
			return nil, true
		}
		out, _ := m.context.RunCommandImmediate(jj.FilesInRevision(rev))
		return common.FileSearch(m.context.CurrentRevset, m.previewModel.Visible(), rev, out), true

	// --- Preview controls ---
	case intents.PreviewToggle:
		var closeCmd tea.Cmd
		restorePreview := m.secondaryRestoreOnClose == secondaryPanePreview
		if m.bookmarkVisible() {
			closeCmd = m.closeBookmarkPane()
			if restorePreview {
				return tea.Batch(closeCmd, common.SelectionChanged(m.context.SelectedItem)), true
			}
		}
		m.togglePreview()
		return tea.Batch(closeCmd, common.SelectionChanged(m.context.SelectedItem)), true
	case intents.PreviewToggleBottom:
		var closeCmd tea.Cmd
		restorePreview := m.secondaryRestoreOnClose == secondaryPanePreview
		if m.bookmarkVisible() {
			closeCmd = m.closeBookmarkPane()
			if restorePreview {
				previewPos := m.previewModel.AtBottom()
				m.previewModel.SetPosition(false, !previewPos)
				return closeCmd, true
			}
		}
		previewPos := m.previewModel.AtBottom()
		m.previewModel.SetPosition(false, !previewPos)
		if m.previewVisible() {
			return closeCmd, true
		}
		m.showPreview()
		return tea.Batch(closeCmd, common.SelectionChanged(m.context.SelectedItem)), true
	case intents.PreviewExpand:
		if !m.previewVisible() {
			return nil, true
		}
		if m.previewSplit != nil && m.previewSplit.State != nil {
			m.previewSplit.State.Expand(config.Current.Preview.WidthIncrementPercentage)
		}
		return nil, true
	case intents.PreviewShrink:
		if !m.previewVisible() {
			return nil, true
		}
		if m.previewSplit != nil && m.previewSplit.State != nil {
			m.previewSplit.State.Shrink(config.Current.Preview.WidthIncrementPercentage)
		}
		return nil, true
	case intents.PreviewScroll:
		if !m.previewVisible() {
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

	// --- Delegated intents ---
	case intents.DiffShow:
		if m.diff == nil {
			m.diff = diff.New("")
		}
		return m.diff.Update(intent), true
	case intents.PreviewShow:
		if m.bookmarkVisible() {
			closeCmd := m.closeBookmarkPane()
			m.showPreview()
			return tea.Batch(closeCmd, m.previewModel.Update(intent)), true
		}
		if !m.previewVisible() {
			m.showPreview()
		}
		return m.previewModel.Update(intent), true

	// --- Status ---
	case intents.ExpandStatusToggle:
		m.status.ToggleStatusExpand()
		return nil, true
	case intents.ToggleBookmarkView:
		if m.bookmarkVisible() {
			return m.closeBookmarkPane(), true
		}
		return m.openBookmarkPane(), true
	case intents.FocusNextPane:
		if !m.bookmarkVisible() {
			return nil, true
		}
		m.focusNextPane()
		return nil, true
	}

	return nil, false
}

func (m *Model) bookmarkPaneScopes() []dispatch.Scope {
	if !m.bookmarkVisible() || !m.bookmarkPaneFocused {
		return nil
	}

	name := actions.ScopeBookmarkView
	if m.bookmarkEditing() {
		name = actions.ScopeBookmarkViewFilter
	}

	return []dispatch.Scope{
		{
			Name:    keybindings.ScopeName(name),
			Leak:    dispatch.LeakNone,
			Handler: m,
		},
	}
}

func (m *Model) handleBookmarkPaneIntent(intent intents.Intent) (tea.Cmd, bool) {
	if !m.bookmarkVisible() || !m.bookmarkPaneFocused {
		return nil, false
	}

	switch intent.(type) {
	case intents.Apply,
		intents.Cancel,
		intents.BookmarkViewNavigate,
		intents.BookmarkViewOpenFilter,
		intents.BookmarkViewToggleExpand,
		intents.BookmarkViewEdit,
		intents.BookmarkViewNew,
		intents.BookmarkViewRename,
		intents.BookmarkViewDelete,
		intents.BookmarkViewForget,
		intents.BookmarkViewTrack,
		intents.BookmarkViewUntrack,
		intents.BookmarkViewMove,
		intents.BookmarkViewReveal,
		intents.BookmarkViewRevealInRevisions,
		intents.BookmarkViewToggleSelect:
	default:
		return nil, false
	}

	if _, ok := intent.(intents.Cancel); ok {
		switch {
		case m.flash.Any():
			m.flash.DeleteOldest()
			return nil, true
		case m.status.StatusExpanded():
			m.status.ToggleStatusExpand()
			return nil, true
		}
	}

	return m.bookmarkPane.HandleIntent(intent)
}

func luaCmd(script string) tea.Cmd {
	return func() tea.Msg {
		return common.RunLuaScriptMsg{Script: script}
	}
}

func (m *Model) stackedScope() (keybindings.ScopeName, bool) {
	if m.stacked == nil {
		return "", false
	}
	scopes := m.stacked.Scopes()
	if len(scopes) == 0 || scopes[0].Name == "" {
		return "", false
	}
	return scopes[0].Name, true
}

func (m *Model) commandHistoryOpen() bool {
	scope, ok := m.stackedScope()
	return ok && scope == actions.ScopeCommandHistory
}

func (m *Model) updateBlockingScope(scope dispatch.Scope, msg tea.KeyMsg) tea.Cmd {
	if scope.Handler == m {
		return nil
	}
	if scope.Handler == m.revsetModel {
		m.state = common.Loading
	}
	return scope.Handler.Update(msg)
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
	revision := target
	if revision == "" {
		revision = commitID
	}
	if revision == "" {
		return nil
	}
	if m.bookmarkRevsetRestore == "" {
		m.bookmarkRevsetRestore = m.context.CurrentRevset
	}
	m.bookmarkRevsetApplied = fmt.Sprintf("::%s", revision)
	return common.UpdateRevSet(m.bookmarkRevsetApplied)
}

func NewUI(c *context.MainContext) *Model {
	revisionsModel := revisions.New(c)
	statusModel := status.New(c)
	flashView := flash.New()
	previewModel := preview.New(c)
	revsetModel := revset.New(c)
	bookmarkPaneModel := bookmarkpane.NewModel(c)

	ui := &Model{
		context:      c,
		state:        common.Loading,
		revisions:    revisionsModel,
		previewModel: previewModel,
		status:       statusModel,
		revsetModel:  revsetModel,
		flash:        flashView,
		bookmarkPane: bookmarkPaneModel,
	}
	ui.initResolver()
	ui.initSplit()
	return ui
}

func (m *Model) setSequenceStatusHelp(continuations []dispatch.Continuation) {
	entries := help.BuildFromContinuations(continuations)
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
	m.resolver = dispatch.NewResolver(dispatcher)
}

func New(c *context.MainContext) tea.Model {
	return &wrapper{ui: NewUI(c)}
}
