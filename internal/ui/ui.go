package ui

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/scripting"
	"github.com/idursun/jjui/internal/ui/actionmeta"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/flash"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/password"
	"github.com/idursun/jjui/internal/ui/render"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
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
	revisions        *revisions.Model
	oplog            *oplog.Model
	revsetModel      *revset.Model
	previewModel     *preview.Model
	diff             *diff.Model
	flash            *flash.Model
	state            common.State
	status           *status.Model
	password         *password.Model
	context          *context.MainContext
	scriptRunner     *scripting.Runner
	sequenceHelp     []help.Entry
	sequenceAutoOpen bool
	resolver         *dispatch.Resolver
	stacked          common.StackedModel
	displayContext   *render.DisplayContext
	width            int
	height           int
	revisionsSplit   *split
	activeSplit      *split

	// mode2031Supported is set when the terminal confirms it supports
	// mode 2031 push. Once true, the OSC 11 polling loop stops.
	mode2031Supported bool
}

type triggerAutoRefreshMsg struct{}

const scopeUi keybindings.ScopeName = "ui"

// colorSchemePollInterval is how often to poll the terminal for its
// current light/dark mode.
var colorSchemePollInterval = time.Second

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
		// reply from initial ansi.RequestModeLightDark check
		if msg.Mode == ansi.ModeLightDark && !msg.Value.IsNotRecognized() {
			m.mode2031Supported = true
		}
		return nil
	case tea.FocusMsg:
		if m.state == common.Ready {
			return common.RefreshAndKeepSelections
		}
		return nil
	case tea.ResumeMsg:
		// common.Suspend disables mode 2031 on the way out, reenable it on resume.
		// Also re-query the background color in case the theme changed while suspended.
		return tea.Batch(
			tea.Raw(ansi.SetModeLightDark),
			tea.RequestBackgroundColor,
		)
	case uv.DarkColorSchemeEvent:
		return m.applyColorScheme(true)
	case uv.LightColorSchemeEvent:
		return m.applyColorScheme(false)
	case tea.BackgroundColorMsg:
		return m.applyColorScheme(msg.IsDark())
	case colorSchemePollTickMsg:
		if m.mode2031Supported {
			return nil
		}
		return tea.Batch(tea.RequestBackgroundColor, scheduleColorSchemePoll())
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
			scopes := m.dispatchScopes()
			cmd, _ := common.RouteIntent(scopes, result.Intent)
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
		model := input.NewWithTitle(msg.Title, msg.Prompt, msg.Value)
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

	if m.diff != nil {
		m.renderDiffLayout(box)
	} else {
		if m.previewModel.Visible() {
			if m.previewModel.AutoPosition() {
				atBottom := m.height >= m.width/2
				m.previewModel.SetPosition(true, atBottom)
			}
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
	if m.revisionsSplit == nil {
		return
	}
	m.revisionsSplit.Primary = primary
	m.revisionsSplit.Secondary = m.previewModel
	m.revisionsSplit.ViewRect(m.displayContext, box)
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

func (m *Model) dispatchScopes() []common.Scope {
	var scopes []common.Scope

	if m.password != nil {
		scopes = append(scopes, m.password.Scopes()...)
	}

	scopes = append(scopes, m.status.Scopes()...)
	if m.revsetModel.IsEditing() {
		scopes = append(scopes, m.revsetModel.Scopes()...)
	}

	if m.diff != nil {
		scopes = append(scopes, m.diff.Scopes()...)
	}

	if m.stacked != nil {
		scopes = append(scopes, m.stacked.Scopes()...)
	} else if m.oplog != nil {
		scopes = append(scopes, m.oplog.Scopes()...)
	} else {
		scopes = append(scopes, m.revisions.Scopes()...)
	}

	if !m.revsetModel.IsEditing() {
		scopes = append(scopes, m.revsetModel.Scopes()...)
	}
	scopes = append(scopes, m.previewModel.Scopes()...)
	scopes = append(scopes, common.Scope{
		Name:    scopeUi,
		Leak:    common.LeakNone,
		Global:  true,
		Handler: m,
	})

	return scopes
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {

	// --- Quit / Suspend ---
	case intents.Quit:
		return common.Quit(), true
	case intents.Suspend:
		return common.Suspend(), true

	// --- Cancel fallback (only reached if no inner scope handled it) ---
	case intents.Cancel:
		if m.flash.Any() {
			m.flash.DeleteOldest()
			return nil, true
		}
		if m.stacked != nil || m.diff != nil || m.oplog != nil {
			return common.Close, true
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
		m.previewModel.ToggleVisible()
		return common.SelectionChanged(m.context.SelectedItem), true
	case intents.PreviewToggleBottom:
		previewPos := m.previewModel.AtBottom()
		m.previewModel.SetPosition(false, !previewPos)
		if m.previewModel.Visible() {
			return nil, true
		}
		m.previewModel.ToggleVisible()
		return common.SelectionChanged(m.context.SelectedItem), true
	case intents.PreviewExpand:
		if !m.previewModel.Visible() {
			return nil, true
		}
		if m.revisionsSplit != nil && m.revisionsSplit.State != nil {
			m.revisionsSplit.State.Expand(config.Current.Preview.WidthIncrementPercentage)
		}
		return nil, true
	case intents.PreviewShrink:
		if !m.previewModel.Visible() {
			return nil, true
		}
		if m.revisionsSplit != nil && m.revisionsSplit.State != nil {
			m.revisionsSplit.State.Shrink(config.Current.Preview.WidthIncrementPercentage)
		}
		return nil, true

	// --- Delegated intents ---
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

	// --- Status ---
	case intents.ExpandStatusToggle:
		m.status.ToggleStatusExpand()
		return nil, true
	}

	return nil, false
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
	frameTickMsg           struct{}
	colorSchemePollTickMsg struct{}
	wrapper                struct {
		ui                 *Model
		scheduledNextFrame bool
		render             bool
		cachedFrame        string
	}
)

func (w *wrapper) Init() tea.Cmd {
	return tea.Batch(
		w.ui.Init(),
		// Enable mode 2031 push notifications and probe whether the terminal
		// supports it. If supported, the mode request returns a tea.ModeReportMsg
		// with ansi.ModeLightDark, and uv.Light/DarkColorSchemeEvent are delivered
		// on OS theme change.
		tea.Raw(ansi.SetModeLightDark),
		tea.Raw(ansi.RequestModeLightDark),
		// Start OSC 11 polling as a baseline for light/dark mode detection
		scheduleColorSchemePoll(),
	)
}

// scheduleColorSchemePoll returns a Cmd that fires a colorSchemePollTickMsg
// after colorSchemePollInterval.
func scheduleColorSchemePoll() tea.Cmd {
	return tea.Tick(colorSchemePollInterval, func(time.Time) tea.Msg {
		return colorSchemePollTickMsg{}
	})
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
	if config.Current.UI.SetWindowTitle {
		v.WindowTitle = fmt.Sprintf("jjui - %s", w.ui.context.Location)
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
	previewModel := preview.New(c)
	revsetModel := revset.New(c)

	ui := &Model{
		context:      c,
		state:        common.Loading,
		revisions:    revisionsModel,
		previewModel: previewModel,
		status:       statusModel,
		revsetModel:  revsetModel,
		flash:        flashView,
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

// applyColorScheme reloads the palette when the terminal's color scheme
// changes (that is, the OS switched between light and dark mode).
func (m *Model) applyColorScheme(isDark bool) tea.Cmd {
	if isDark == m.context.TerminalHasDarkBackground {
		return nil
	}
	scheme := "light"
	if isDark {
		scheme = "dark"
	}
	log.Printf("color scheme changed to %s", scheme)
	m.context.TerminalHasDarkBackground = isDark
	theme, err := config.ResolveTheme(isDark, m.context.JJConfig.GetApplicableColors())
	if err != nil {
		log.Printf("failed to resolve %s theme: %v", scheme, err)
		return nil
	}
	common.DefaultPalette.Update(theme)
	return func() tea.Msg { return common.ThemeChangedMsg{} }
}

func New(c *context.MainContext) tea.Model {
	return &wrapper{ui: NewUI(c)}
}
