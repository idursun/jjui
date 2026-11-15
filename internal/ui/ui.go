package ui

import (
	"fmt"
	"log"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/idursun/jjui/internal/ui/theme"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/scripting"
	"github.com/idursun/jjui/internal/ui/actionmeta"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/annotation"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/choose"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/exec_process"
	"github.com/idursun/jjui/internal/ui/exec_prompt"
	"github.com/idursun/jjui/internal/ui/flash"
	"github.com/idursun/jjui/internal/ui/fuzzy_files"
	"github.com/idursun/jjui/internal/ui/help"
	"github.com/idursun/jjui/internal/ui/input"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations/target_picker"
	"github.com/idursun/jjui/internal/ui/oplog"
	"github.com/idursun/jjui/internal/ui/password"
	"github.com/idursun/jjui/internal/ui/quick_search"
	"github.com/idursun/jjui/internal/ui/redo"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/internal/ui/split"
	"github.com/idursun/jjui/internal/ui/status"
	"github.com/idursun/jjui/internal/ui/undo"
)

type Model struct {
	revisions        *revisions.Model
	layers           []layer
	revsetModel      *revset.Model
	flash            *flash.Model
	state            common.State
	status           *status.Model
	password         *password.Model
	context          *context.MainContext
	scriptRunners    []scriptFrame
	queuedScripts    []string
	sequenceHelp     []help.Entry
	sequenceAutoOpen bool
	resolver         *dispatch.Resolver
	displayContext   *render.DisplayContext
	frameCursor      *tea.Cursor
	lastHighlighted  common.SelectedItem
	width            int
	height           int
	splitContainer   *split.SplitContainer

	// mode2031Supported is set when the terminal confirms it supports
	// mode 2031 push. Once true, the OSC 11 polling loop stops.
	mode2031Supported bool
}

var _ common.ApplicationStateProvider = (*Model)(nil)

type scriptFrame struct {
	runner       *scripting.Runner
	completionID string
}

type triggerAutoRefreshMsg struct{}

const (
	scopeUi      keybindings.ScopeName = "ui"
	scopeOverlay keybindings.ScopeName = "ui.overlay"
)

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.revisions.Init(), m.scheduleAutoRefresh())
}

func (m *Model) Selection() common.SelectionSnapshot {
	var providers []common.SelectionProvider
	for _, l := range slices.Backward(m.layers) {
		if provider, ok := l.model.(common.SelectionProvider); ok {
			providers = append(providers, provider)
		}
	}
	if m.revisions != nil {
		providers = append(providers, m.revisions)
	}
	var snapshot common.SelectionSnapshot
	for _, provider := range providers {
		part := provider.Selection()
		if snapshot.Highlighted == nil && part.Highlighted != nil {
			snapshot.Highlighted = part.Highlighted
		}
		snapshot.Checked = append(snapshot.Checked, part.Checked...)
		snapshot.CheckedObjects = append(snapshot.CheckedObjects, part.CheckedObjects...)
	}
	if rev, ok := snapshot.Highlighted.(common.SelectedRevision); ok {
		if kind, value, name, focused := m.revsetModel.FocusedRemote(); focused {
			rev.ObjectKind, rev.ObjectValue = kind, value
			rev.ObjectText, rev.ObjectRemote, rev.ObjectConflicted, rev.ObjectRemotes = name, "", false, nil
			snapshot.Highlighted = rev
		}
	}
	return snapshot
}

// QueryState resolves public binding and Lua state paths against retained model owners.
// Focus and visibility do not participate in lookup; owners decide whether a
// model is still live and can answer the local property.
func (m *Model) QueryState(name string) (any, bool) {
	if after, ok := strings.CutPrefix(name, "ui.preview."); ok {
		switch after {
		case "can_expand":
			if m.splitContainer == nil {
				return false, true
			}
			return m.splitContainer.CanResize(config.Current.Preview.WidthIncrementPercentage), true
		case "can_shrink":
			if m.splitContainer == nil {
				return false, true
			}
			return m.splitContainer.CanResize(-config.Current.Preview.WidthIncrementPercentage), true
		}
	}
	if annotationModel, ok := findLayer[*annotation.Model](m); ok && strings.HasPrefix(name, "annotation.") {
		return annotationModel.QueryState(strings.TrimPrefix(name, "annotation."))
	}

	if after, ok := strings.CutPrefix(name, "revset."); ok {
		return m.revsetModel.QueryState(after)
	}
	if strings.HasPrefix(name, "revisions.") && m.revisions != nil {
		return m.revisions.QueryState(strings.TrimPrefix(name, "revisions."))
	}
	return nil, false
}

func (m *Model) Update(msg tea.Msg) (cmd tea.Cmd) {
	var completionID string
	defer func() {
		highlighted := m.Selection().Highlighted
		unchanged := highlighted == nil && m.lastHighlighted == nil ||
			highlighted != nil && m.lastHighlighted != nil && highlighted.Equal(m.lastHighlighted)
		if !unchanged {
			m.lastHighlighted = highlighted
			cmd = tea.Batch(cmd, common.SelectionChanged(highlighted))
		}
		// Dispatched actions must finish their commands and selection notification
		// before Lua resumes. Batching around the completion sequence would race it.
		cmd = tea.Sequence(cmd, actionCompleted(completionID))
	}()

	if closeMsg, ok := msg.(common.CloseViewMsg); ok {
		// The expanded status help is an overlay over the active view. When there
		// is no dialog, close the overlay before closing that view.
		if !closeMsg.Applied && m.dialog() == nil && m.status != nil && m.status.StatusExpanded() {
			m.status.SetStatusExpanded(false)
			return nil
		}
		if len(m.layers) > 0 {
			return m.closeTopLayer(closeMsg)
		}
		if m.splitContainer.ContentFocused() && m.splitContainer.Close() {
			return nil
		}
	}

	var cmds []tea.Cmd

	if cmd, handled := m.handleSplitMsg(msg); handled {
		return cmd
	} else if cmd != nil {
		cmds = append(cmds, cmd)
	}

	if cmd, handled := m.handleAppearanceMsg(msg); handled {
		return cmd
	}

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
	case tea.MouseReleaseMsg, tea.MouseMotionMsg:
		// Only the split separator drag follows the mouse; clicks and wheel
		// events go through the frame's interactions below.
		m.handleSplitMouseMsg(msg)
		return nil
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
			if m.sequenceHelp != nil {
				m.sequenceHelp = nil
				if m.sequenceAutoOpen {
					m.status.SetStatusExpanded(false)
				}
				m.sequenceAutoOpen = false
			}
			if result.LuaScript != "" {
				return luaCmd(result.LuaScript)
			}
			if result.Intent != nil {
				start := slices.IndexFunc(scopes, func(scope common.Scope) bool {
					return string(scope.Name) == result.Scope
				})
				if start < 0 {
					return nil
				}
				if cmd, handled := common.RouteIntent(scopes[start:], result.Intent); handled {
					return cmd
				}
				if scopes[start].Leak != common.LeakAll {
					return m.updateBlockingScope(scopes[start], msg)
				}
				return nil
			}
			if result.Consumed {
				return nil
			}

			for _, scope := range scopes {
				if scope.Leak != common.LeakAll {
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
		if msg.Err != nil {
			cmds = append(cmds, m.openDialog(exec_prompt.New(m.context, msg.Msg.Mode, msg.Msg.Line)))
		}
	case common.UpdateRevisionsSuccessMsg:
		m.state = common.Ready
	case triggerAutoRefreshMsg:
		return tea.Batch(m.scheduleAutoRefresh(), func() tea.Msg {
			return common.AutoRefreshMsg{}
		})
	case common.UpdateRevSetMsg:
		if m.revsetModel.IsStaleRevset(string(msg)) {
			return nil
		}
		m.context.CurrentRevset = string(msg)
		if m.context.CurrentRevset == "" {
			m.context.CurrentRevset = m.context.DefaultRevset
		}
		m.revsetModel.AddToHistory(m.context.CurrentRevset)
		m.revsetModel.Update(msg)
		return common.Refresh
	case common.RunLuaScriptMsg:
		if msg.CompletionID == "" && len(m.scriptRunners) > 0 {
			if len(m.queuedScripts) >= maxQueuedScripts {
				err := fmt.Errorf("lua script is already running")
				return intents.Invoke(intents.AddMessage{Text: err.Error(), Err: err})
			}
			m.queuedScripts = append(m.queuedScripts, msg.Script)
			return nil
		}
		return m.runScript(msg.Script, msg.CompletionID)
	case common.DispatchActionMsg:
		if m.resolver == nil {
			return actionCompleted(msg.CompletionID)
		}
		actionName := strings.TrimSpace(msg.Action)
		if actionmeta.IsBuiltInAction(actionName) {
			if err := actionmeta.ValidateBuiltInActionArgs(actionName, msg.Args); err != nil {
				return tea.Sequence(
					intents.Invoke(intents.AddMessage{Text: err.Error(), Err: err}),
					actionCompleted(msg.CompletionID),
				)
			}
		}
		action := keybindings.Action(actionName)
		var result dispatch.Result
		if msg.BuiltIn {
			result = m.resolver.ResolveBuiltInAction(action, msg.Args)
		} else {
			result = m.resolver.ResolveAction(action, msg.Args)
		}
		if result.LuaScript != "" {
			return luaCmdWithCompletion(result.LuaScript, msg.CompletionID)
		}
		if result.Intent != nil {
			scopes := m.dispatchScopes()
			cmd, _ := common.RouteIntent(scopes, result.Intent)
			completionID = msg.CompletionID
			return cmd
		}
		return actionCompleted(msg.CompletionID)
	case common.ShowChooseMsg:
		return m.openDialog(choose.NewWithOptions(msg.Options, msg.Title, msg.Ordered))
	case choose.SelectedMsg, choose.CancelledMsg:
		m.closeDialog()
	case common.ShowInputMsg:
		return m.openDialog(input.NewWithTitle(msg.Title, msg.Prompt, msg.Value))
	case input.SelectedMsg, input.CancelledMsg:
		m.closeDialog()
	case common.OpenTargetPickerMsg:
		if m.acceptsTargets() {
			return m.openDialog(target_picker.NewModel(m.context, msg.Payload, msg.Sources...))
		}
	case target_picker.TargetSelectedMsg:
		if m.acceptsTargets() && m.dialog() != nil {
			m.closeDialog()
			return m.screen().Update(msg)
		}
	case target_picker.TargetPickerCancelMsg:
		if m.acceptsTargets() && m.dialog() != nil {
			m.closeDialog()
			return nil
		}
	case common.TogglePasswordMsg:
		if msg.Password == nil {
			if m.password != nil && m.password.ID() == msg.ID {
				// let the current prompt clean itself
				m.password.Update(msg)
				m.password = nil
			}
		} else {
			if m.password != nil {
				// Close the superseded prompt's response channel.
				m.password.Update(common.TogglePasswordMsg{ID: m.password.ID()})
			}
			// overwrite current prompt. This can happen for ssh-sk keys:
			//   - first prompt reads "Confirm user presence for ..."
			//   - if the user denies the request on the device, a new prompt automatically happen "Enter PIN for ...
			m.password = password.New(msg)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.context.SetTerminalSize(msg.Width, msg.Height)
	}

	// Keys and mouse events have returned above. Every other message is
	// broadcast to all models.
	primary := m.primary()
	cmds = append(cmds, m.revsetModel.Update(msg))
	cmds = append(cmds, m.flash.Update(msg))
	for _, l := range m.layers {
		if l.model != primary {
			cmds = append(cmds, l.model.Update(msg))
		}
	}

	if len(m.scriptRunners) > 0 {
		top := len(m.scriptRunners) - 1
		frame := m.scriptRunners[top]
		cmd := frame.runner.HandleMsg(msg)
		if frame.runner.Done() {
			m.scriptRunners = m.scriptRunners[:top]
			cmd = tea.Sequence(cmd, actionCompleted(frame.completionID), m.runQueuedScripts())
		}
		cmds = append(cmds, cmd)
	}

	cmds = append(cmds, primary.Update(msg))

	cmds = append(cmds, m.updateSplit(msg))

	return tea.Batch(cmds...)
}

func (m *Model) updateStatus() {
	if m.sequenceHelp != nil && m.resolver != nil {
		m.sequenceHelp = help.BuildFromContinuations(m.resolver.Continuations(m.dispatchScopes()))
		if m.sequenceHelp == nil {
			m.sequenceHelp = []help.Entry{}
		}
	}
	m.status.SetCheckedCount(m.revisions.CheckedCount())
	m.status.Sync(m.dispatchScopes(), m.sequenceHelp)
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	m.displayContext = render.NewDisplayContext()

	m.updateStatus()

	box := layout.NewBox(layout.Rect(0, 0, m.width, m.height))
	screenBuf := render.NewScreenBuffer(m.width, m.height)

	switch screen := m.screen().(type) {
	case nil:
		m.updateSplitAutoPosition()
		content, status := m.splitFooter(box)
		rows := content.V(layout.Fixed(m.revsetModel.Height()), layout.Fill(1))
		m.revsetModel.ViewRect(m.displayContext, rows[0])
		m.renderSplit(m.revisions, rows[1])
		m.renderFooter(status)
	case *oplog.Model:
		m.updateSplitAutoPosition()
		m.renderWithStatus(box, func(content layout.Box) {
			m.renderSplit(screen, content)
		})
	default:
		m.renderWithStatus(box, func(content layout.Box) {
			screen.ViewRect(m.displayContext, content)
		})
	}

	for _, l := range m.layers[m.screenIndex()+1:] {
		l.model.ViewRect(m.displayContext, box)
	}

	if m.password == nil {
		if scope, ok := m.stackedScope(); !ok || scope != actions.ScopeCommandHistory {
			flashBox := box
			if box.R.Dy() > 0 {
				flashBox, _ = box.CutBottom(1)
			}
			m.flash.ViewRect(m.displayContext, flashBox)
		}
	}

	m.splitContainer.RenderOverlay(m.displayContext, box)

	if m.password != nil {
		m.password.ViewRect(m.displayContext, box)
	}

	m.displayContext.Render(screenBuf)
	finalView := screenBuf.Render()
	m.frameCursor = m.displayContext.Cursor()
	return strings.ReplaceAll(finalView, "\r", "")
}

func (m *Model) renderWithStatus(box layout.Box, renderContent func(layout.Box)) {
	content, status := m.splitFooter(box)
	renderContent(content)
	m.renderFooter(status)
}

func (m *Model) splitFooter(box layout.Box) (content, status layout.Box) {
	if box.R.Dy() == 0 {
		return box, status
	}
	return box.CutBottom(1)
}

func (m *Model) renderFooter(status layout.Box) {
	if status.R.Dy() > 0 {
		m.status.ViewRect(m.displayContext, status)
	}
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

func (m *Model) dispatchScopes() []common.Scope {
	var scopes []common.Scope

	if m.password != nil {
		scopes = append(scopes, m.password.Scopes()...)
	}

	// Error and sticky flash messages and the expanded help are drawn over
	// every view, so esc closes them before it reaches the view underneath.
	// Messages that expire on their own, and text inputs, keep esc.
	if (m.flashVisible() || m.status.StatusExpanded()) && !m.inputFocused() {
		scopes = append(scopes, common.Scope{
			Name:         scopeOverlay,
			Leak:         common.LeakAll,
			HideFromHelp: true,
			Handler:      m,
		})
	}

	if m.revsetModel.IsEditing() {
		scopes = append(scopes, m.revsetModel.Scopes()...)
	}

	// Only the top layer receives keys. The split panes beside the primary
	// view (preview, bookmark pane) join only while the primary view is on top.
	primary := m.primary()
	if top, ok := m.topLayer(); ok && top.model != primary {
		scopes = append(scopes, top.model.Scopes()...)
	} else {
		content := m.splitScopes(primary.Scopes())
		// The bookmarks view's keys go just before the revisions scope, which
		// also binds tab, / and esc. A focused remotes line goes first so the
		// log's keys stay away from it.
		if bookmarks := m.revsetModel.BookmarksScopes(); m.revisions.InNormalMode() && len(bookmarks) > 0 {
			i := 0
			if !m.revsetModel.RemotesFocused() {
				if j := slices.IndexFunc(content, func(s common.Scope) bool { return s.Name == actions.ScopeRevisions }); j >= 0 {
					i = j
				}
			}
			content = slices.Concat(content[:i], bookmarks, content[i:])
		}
		scopes = append(scopes, content...)
	}

	if !m.revsetModel.IsEditing() {
		scopes = append(scopes, m.revsetModel.Scopes()...)
	}
	scopes = append(scopes, common.Scope{
		Name:    scopeUi,
		Leak:    common.LeakNone,
		Global:  true,
		Handler: m,
	})

	return scopes
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	if cmd, handled := m.handleSplitIntent(intent); handled {
		return cmd, true
	}

	switch intent := intent.(type) {

	// --- Quit / Suspend ---
	case intents.Quit:
		return common.Quit(), true
	case intents.Suspend:
		return common.Suspend(), true
	case intents.ChangeTheme:
		return m.changeTheme(intent), true

	// --- Cancel fallback (only reached if no inner scope handled it) ---
	case intents.Cancel:
		if m.flash.Any() {
			m.flash.DeleteOldest()
			return nil, true
		}
		if len(m.layers) > 0 {
			return common.Close, true
		}
		if m.status.StatusExpanded() {
			m.status.ToggleStatusExpand()
			return nil, true
		}
		return nil, false

	// --- Open stacked views ---
	case intents.OpLogOpen:
		model := oplog.New(m.context)
		m.openScreen(model)
		return model.Init(), true
	case intents.Undo:
		return m.openDialog(undo.NewModel(m.context)), true
	case intents.Redo:
		return m.openDialog(redo.NewModel(m.context)), true
	case intents.OpenHelp:
		// Help opens only over the revisions view or the oplog.
		if top, ok := m.topLayer(); ok && top.model != m.primary() {
			return nil, true
		}
		return m.openDialog(help.New()), true
	case intents.CommandHistoryToggle:
		if scope, ok := m.stackedScope(); ok && scope == actions.ScopeCommandHistory {
			m.closeDialog()
			return nil, true
		}
		return m.openDialog(m.flash.NewHistory()), true

	// --- Activate input modes ---
	case intents.Edit:
		return m.revsetModel.Update(intent), true
	case intents.ExecJJ:
		return m.openDialog(exec_prompt.New(m.context, common.ExecJJ, "")), true
	case intents.ExecShell:
		return m.openDialog(exec_prompt.New(m.context, common.ExecShell, "")), true
	case intents.QuickSearch:
		return m.openDialog(quick_search.New(m.context)), true
	case intents.FileSearchToggle:
		rev := m.revisions.SelectedRevision()
		if rev == nil {
			return nil, true
		}
		return m.openDialog(fuzzy_files.New(m.context, m.context.CurrentRevset, rev)), true

	// --- Delegated intents ---
	case intents.DiffShow:
		model, ok := findLayer[*diff.Model](m)
		if !ok {
			model = diff.NewWithContext(m.context, "", nil)
		}
		m.openScreen(model)
		return model.Update(intent), true
	case intents.AnnotationShow:
		if intent.ChangeID == "" {
			return nil, true
		}
		model := annotation.NewRange(m.context, intent.From, intent.ChangeID)
		m.openScreen(model)
		return model.Init(), true
	// --- Status ---
	case intents.DismissOverlay:
		if m.flashVisible() {
			m.flash.DeleteOldestPersistent()
		} else if m.status.StatusExpanded() {
			m.status.SetStatusExpanded(false)
		}
		return nil, true
	case intents.ExpandStatusToggle:
		m.status.ToggleStatusExpand()
		return nil, true
	}

	return nil, false
}

// maxQueuedScripts bounds key presses waiting behind a script that may never
// finish, such as one waiting for a view to close.
const maxQueuedScripts = 8

func (m *Model) runScript(script string, completionID string) tea.Cmd {
	runner, cmd, err := scripting.RunScript(m.context, script)
	if err != nil {
		return tea.Sequence(func() tea.Msg {
			return common.CommandCompletedMsg{Err: err}
		}, actionCompleted(completionID))
	}
	if runner != nil && !runner.Done() {
		m.scriptRunners = append(m.scriptRunners, scriptFrame{
			runner:       runner,
			completionID: completionID,
		})
		return cmd
	}
	return tea.Sequence(cmd, actionCompleted(completionID))
}

// runQueuedScripts starts queued scripts in order until one has to wait. It
// runs in the same update that finished the previous script, so a new key
// press cannot start ahead of the queue.
func (m *Model) runQueuedScripts() tea.Cmd {
	var cmds []tea.Cmd
	for len(m.scriptRunners) == 0 && len(m.queuedScripts) > 0 {
		script := m.queuedScripts[0]
		m.queuedScripts = m.queuedScripts[1:]
		cmds = append(cmds, m.runScript(script, ""))
	}
	return tea.Sequence(cmds...)
}

func luaCmd(script string) tea.Cmd {
	return luaCmdWithCompletion(script, "")
}

func luaCmdWithCompletion(script string, completionID string) tea.Cmd {
	return func() tea.Msg {
		return common.RunLuaScriptMsg{Script: script, CompletionID: completionID}
	}
}

func actionCompleted(id string) tea.Cmd {
	if id == "" {
		return nil
	}
	return func() tea.Msg {
		return common.ActionCompletedMsg{ID: id}
	}
}

// inputFocused reports whether the model that receives keys is taking text,
// following the same order as dispatchScopes.
func (m *Model) inputFocused() bool {
	if m.revsetModel.IsEditing() {
		return true
	}
	var receiver any = m.primary()
	if top, ok := m.topLayer(); ok && top.model != m.primary() {
		receiver = top.model
	} else if m.splitContainer != nil && m.splitContainer.ContentFocused() {
		return m.splitContainer.ContentEditing()
	}
	editable, ok := receiver.(common.Editable)
	return ok && editable.IsEditing()
}

func (m *Model) flashVisible() bool {
	scope, ok := m.stackedScope()
	return m.flash.AnyPersistent() && (!ok || scope != actions.ScopeCommandHistory)
}

func (m *Model) stackedScope() (keybindings.ScopeName, bool) {
	dialog := m.dialog()
	if dialog == nil {
		return "", false
	}
	scopes := dialog.Scopes()
	if len(scopes) == 0 || scopes[0].Name == "" {
		return "", false
	}
	return scopes[0].Name, true
}

func (m *Model) updateBlockingScope(scope common.Scope, msg tea.KeyMsg) tea.Cmd {
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
		cachedCursor       *tea.Cursor
	}
)

func (w *wrapper) Init() tea.Cmd {
	return tea.Batch(
		w.ui.Init(),
		initTerminalAppearance(theme.DefaultPalette.BackgroundBlend()),
	)
}

func (w *wrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(frameTickMsg); ok {
		w.render = true
		w.scheduledNextFrame = false
		return w, nil
	}
	cmd := w.ui.Update(msg)
	if _, ok := msg.(common.ExecMsg); ok {
		// Force a fresh frame before Bubble Tea releases/restores the terminal
		// for tea.Exec. Otherwise RestoreTerminal can repaint using the last
		// cached cursor-bearing frame from the command prompt.
		w.render = true
		w.scheduledNextFrame = false
		return w, cmd
	}
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
		w.cachedCursor = w.ui.frameCursor
		w.render = false
	}
	v := tea.NewView(w.cachedFrame)
	v.Cursor = w.cachedCursor
	if config.Current.UI.SetWindowTitle != config.WindowTitleOff {
		title := w.ui.context.Location
		if config.Current.UI.SetWindowTitle == config.WindowTitleBase {
			title = filepath.Base(title)
		}
		v.WindowTitle = fmt.Sprintf("jjui - %s", title)
	}
	v.AltScreen = true
	v.ReportFocus = true
	v.MouseMode = tea.MouseModeCellMotion
	if !config.Current.UI.MouseSupport {
		v.MouseMode = tea.MouseModeNone
	}
	return v
}

func NewUI(c *context.MainContext) *Model {
	revisionsModel := revisions.New(c)
	statusModel := status.New(c)
	flashView := flash.New()
	revsetModel := revset.New(c)

	ui := &Model{
		context:     c,
		state:       common.Loading,
		revisions:   revisionsModel,
		status:      statusModel,
		revsetModel: revsetModel,
		flash:       flashView,
	}
	c.SetStateProvider(ui)
	ui.initSplitContainer()
	dispatcher, err := dispatch.NewDispatcher(config.BindingsToRuntime(config.Current.Bindings))
	if err != nil {
		log.Printf("failed to initialise key bindings: %v", err)
		return ui
	}
	dispatcher.SetStateProvider(ui)
	ui.resolver = dispatch.NewResolver(dispatcher)
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

func New(c *context.MainContext) tea.Model {
	return &wrapper{ui: NewUI(c)}
}
