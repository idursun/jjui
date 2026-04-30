package ui

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/scripting"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/git"
	"github.com/idursun/jjui/internal/ui/help"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/describe"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/operations/rebase"
	"github.com/idursun/jjui/internal/ui/operations/set_parents"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dispatchAction(model *Model, action keybindings.Action, args map[string]any) (tea.Cmd, bool) {
	result := model.resolver.ResolveAction(action, args)
	if result.LuaScript != "" {
		return luaCmd(result.LuaScript), true
	}
	if result.Intent != nil {
		scopes := model.dispatchScopes()
		cmd, handled := dispatch.RouteIntent(scopes, result.Intent)
		return cmd, handled
	}
	return nil, result.Consumed
}

func TestWrapperView_SetsWindowTitleWhenEnabled(t *testing.T) {
	origSetWindowTitle := config.Current.UI.SetWindowTitle
	t.Cleanup(func() { config.Current.UI.SetWindowTitle = origSetWindowTitle })
	config.Current.UI.SetWindowTitle = true

	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	ctx.Location = "/tmp/repo"
	w := &wrapper{ui: &Model{context: ctx}, cachedFrame: "frame"}

	view := w.View()

	assert.Equal(t, "jjui - /tmp/repo", view.WindowTitle)
}

func TestWrapperView_LeavesWindowTitleEmptyWhenDisabled(t *testing.T) {
	origSetWindowTitle := config.Current.UI.SetWindowTitle
	t.Cleanup(func() { config.Current.UI.SetWindowTitle = origSetWindowTitle })
	config.Current.UI.SetWindowTitle = false

	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	ctx.Location = "/tmp/repo"
	w := &wrapper{ui: &Model{context: ctx}, cachedFrame: "frame"}

	view := w.View()

	assert.Empty(t, view.WindowTitle)
}

func Test_Update_PreviewScrollKeysWorkWhenVisible(t *testing.T) {
	tests := []struct {
		name           string
		key            tea.KeyPressMsg
		expectedScroll int // positive = down, negative = up
	}{
		{
			name:           "ctrl+d scrolls half page down",
			key:            tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl},
			expectedScroll: 1,
		},
		{
			name:           "ctrl+u scrolls half page up",
			key:            tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl},
			expectedScroll: -1,
		},
		{
			name:           "ctrl+n scrolls down",
			key:            tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl},
			expectedScroll: 1,
		},
		{
			name:           "ctrl+p scrolls up",
			key:            tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl},
			expectedScroll: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			commandRunner := test.NewTestCommandRunner(t)
			ctx := test.NewTestContext(commandRunner)

			model := NewUI(ctx)
			model.previewModel.SetVisible(true)

			var content strings.Builder
			for range 100 {
				content.WriteString("line content here\n")
			}
			model.previewModel.SetContent(content.String())

			// Force internal view port to have a size
			model.previewModel.ViewRect(render.NewDisplayContext(), layout.NewBox(layout.Rect(0, 0, 100, 50)))

			initialYOffset := model.previewModel.YOffset()

			// Send the key message
			model.Update(tc.key)

			newYOffset := model.previewModel.YOffset()
			if tc.expectedScroll > 0 {
				assert.Greater(t, newYOffset, initialYOffset, "expected scroll down for key %s", tc.name)
			} else {
				// For scroll up, we need content scrolled down first
				model.previewModel.Scroll(50) // scroll down first
				scrolledYOffset := model.previewModel.YOffset()
				model.Update(tc.key)
				newYOffset = model.previewModel.YOffset()
				assert.Less(t, newYOffset, scrolledYOffset, "expected scroll up for key %s", tc.name)
			}
		})
	}
}

func Test_Update_PreviewResizeKeysWorkWhenVisible(t *testing.T) {
	tests := []struct {
		name           string
		key            tea.KeyPressMsg
		expectedResize int // positive = expand, negative = shrink
	}{
		{
			name:           "ctrl+l shrinks preview",
			key:            tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl},
			expectedResize: -1,
		},
		{
			name:           "ctrl+h expands preview",
			key:            tea.KeyPressMsg{Code: 'h', Mod: tea.ModCtrl},
			expectedResize: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			commandRunner := test.NewTestCommandRunner(t)
			ctx := test.NewTestContext(commandRunner)

			model := NewUI(ctx)
			model.previewModel.SetVisible(true)

			initialWidth := model.revisionsSplit.State.Percent
			model.Update(tc.key)
			newWidth := model.revisionsSplit.State.Percent

			if tc.expectedResize > 0 {
				assert.Greater(t, newWidth, initialWidth, "expected preview to expand for key %s", tc.name)
			} else {
				assert.Less(t, newWidth, initialWidth, "expected preview to shrink for key %s", tc.name)
			}
		})
	}
}

func Test_UpdateStatus_RevsetEditingShowsRevsetHelp(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()
	ctx := test.NewTestContext(commandRunner)

	model := NewUI(ctx)

	// Activate revset editing
	model.revsetModel.Update(revset.EditRevSetMsg{})
	assert.True(t, model.revsetModel.Editing, "revset should be in editing mode")

	// Trigger status update
	model.updateStatus()
	assert.Equal(t, "revset", model.status.Mode(), "status mode should be 'revset'")
	assert.NotNil(t, model.status.Help(), "status help should be available in revset mode")
}

func Test_UpdateStatus_FlashVisibleShowsHistoryModeAndHelp(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.show_command_history", Scope: "ui", Key: config.StringList{"W"}},
		{Action: "revisions.move_down", Scope: "revisions", Key: config.StringList{"j"}},
		{Action: "command_history.move_down", Scope: "command_history", Key: config.StringList{"j"}},
		{Action: "command_history.delete_selected", Scope: "command_history", Key: config.StringList{"d"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.Update(intents.CommandHistoryToggle{})
	model.updateStatus()

	assert.Equal(t, "history", model.status.Mode())
	entries := help.FlatEntries(model.status.Help())
	require.Len(t, entries, 3)
	assert.Equal(t, "j", entries[0].Label)
	assert.Equal(t, "move down", entries[0].Desc)
	assert.Equal(t, "d", entries[1].Label)
	assert.Equal(t, "delete selected", entries[1].Desc)
	assert.Equal(t, "W", entries[2].Label)
	assert.Equal(t, "show command history", entries[2].Desc)
}

func Test_DispatchScopes_UsesCommandHistoryScopeWhenOpen(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.Update(intents.CommandHistoryToggle{})

	scopes := model.dispatchScopes()
	require.NotEmpty(t, scopes)
	assert.Equal(t, keybindings.ScopeName(actions.ScopeCommandHistory), scopes[0].Name)
}

func Test_HandleDispatchedAction_UsesFlashScopeWhenVisible(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.Update(intents.CommandHistoryToggle{})
	scope, ok := model.stackedScope()
	require.True(t, ok)
	assert.Equal(t, keybindings.ScopeName(actions.ScopeCommandHistory), scope)

	cmd, handled := dispatchAction(model, keybindings.Action("command_history.close"), nil)
	assert.True(t, handled)
	require.NotNil(t, cmd)
	closeMsg, ok := cmd().(common.CloseViewMsg)
	require.True(t, ok)
	model.Update(closeMsg)
	_, ok = model.stackedScope()
	assert.False(t, ok)
}

func TestUndoDialogRawConfirmationKeysStillWork(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLog(1))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	test.SimulateModel(model, func() tea.Msg { return intents.Undo{} })
	scope, ok := model.stackedScope()
	require.True(t, ok)
	assert.Equal(t, keybindings.ScopeName(actions.ScopeUndo), scope)

	test.SimulateModel(model, func() tea.Msg {
		return tea.KeyPressMsg{Text: "n", Code: 'n'}
	})

	_, ok = model.stackedScope()
	assert.False(t, ok, "pressing n should close the undo confirmation")
}

func TestRedoDialogRawConfirmationKeysStillWork(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLog(1))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	test.SimulateModel(model, func() tea.Msg { return intents.Redo{} })
	scope, ok := model.stackedScope()
	require.True(t, ok)
	assert.Equal(t, keybindings.ScopeName(actions.ScopeRedo), scope)

	test.SimulateModel(model, func() tea.Msg {
		return tea.KeyPressMsg{Text: "n", Code: 'n'}
	})

	_, ok = model.stackedScope()
	assert.False(t, ok, "pressing n should close the redo confirmation")
}

// this test verifies that when `git` is activated and `status` is expanded,
// pressing `esc` closes expanded `status`
func Test_GitWithExpandedStatus_EscClosesStackedFirst(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.expand_status", Scope: "ui", Key: config.StringList{"?"}},
		{Action: "ui.cancel", Scope: "ui", Key: config.StringList{"esc"}},
		{Action: "git.move_up", Scope: "git", Key: config.StringList{"k"}},
		{Action: "git.move_down", Scope: "git", Key: config.StringList{"j"}},
		{Action: "git.apply", Scope: "git", Key: config.StringList{"enter"}},
		{Action: "git.push", Scope: "git", Key: config.StringList{"p"}},
		{Action: "git.fetch", Scope: "git", Key: config.StringList{"f"}},
		{Action: "git.filter", Scope: "git", Key: config.StringList{"/"}},
		{Action: "git.cycle_remotes", Scope: "git", Key: config.StringList{"tab"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.Histories = config.NewHistories()
	model := NewUI(ctx)

	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	// Directly set stacked to git model (simulates pressing 'g')
	gitModel := git.NewModel(ctx, jj.NewSelectedRevisions())
	test.SimulateModel(gitModel, gitModel.Init())
	model.stacked = gitModel
	assert.NotNil(t, model.stacked, "stacked (git) should be set")

	// Render to trigger status truncation detection
	_ = model.View()

	// Expand status directly; this test validates esc precedence while git is stacked.
	model.status.SetStatusExpanded(true)
	assert.True(t, model.status.StatusExpanded(), "status should be expanded before pressing esc")

	// Press 'esc' to close stacked first.
	test.SimulateModel(model, test.Press(tea.KeyEscape))
	assert.True(t, model.status.StatusExpanded(), "status should remain expanded while stacked is closed first")

	// Stacked (git) should be closed first
	assert.Nil(t, model.stacked, "stacked (git) should close before expanded status")
}

func Test_Update_GitFilteredShortcutKeysDoNotLeakToRevisions(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte(""))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	gitModel := git.NewModel(ctx, jj.NewSelectedRevisions())
	test.SimulateModel(gitModel, gitModel.Init())
	test.SimulateModel(gitModel, func() tea.Msg { return intents.GitFilter{Kind: intents.GitFilterFetch} })
	model.stacked = gitModel

	key := tea.KeyPressMsg{Text: "a", Code: 'a'}
	result := model.resolver.ResolveKey(key, model.dispatchScopes())
	assert.Nil(t, result.Intent, "git shortcut keys should bypass bound outer scopes and fall through as raw input")
	assert.False(t, result.Consumed, "git shortcut key should remain unbound at resolver level")

	cmd := model.Update(key)
	assert.NotNil(t, cmd, "git model should receive raw shortcut key and produce command")
}

func Test_Update_GlobalBindingsFromConfigOverrideLegacyGlobalKeys(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()

	config.Current.Bindings = []config.BindingConfig{
		{
			Action: "ui.cancel",
			Scope:  "ui",
			Key:    config.StringList{"ctrl+x"},
		},
	}

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	ctx.Histories = config.NewHistories()
	model := NewUI(ctx)

	model.flash.Update(intents.AddMessage{Text: "test error", Err: fmt.Errorf("test")})
	model.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	assert.False(t, model.flash.Any(), "ctrl+x should use configured global cancel binding")

	model.flash.Update(intents.AddMessage{Text: "test error", Err: fmt.Errorf("test")})
	model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.True(t, model.flash.Any(), "esc should not act as global cancel when global bindings are configured")
}

func Test_Update_RevisionsEscClearsCheckedSelections_WithDefaultBindings(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	ctx.AddCheckedItem(common.SelectedRevision{ChangeId: "change-1", CommitId: "commit-1"})
	ctx.AddCheckedItem(common.SelectedRevision{ChangeId: "change-2", CommitId: "commit-2"})

	require.Len(t, ctx.CheckedItems, 2, "setup should create multiple checked revisions")

	cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.Nil(t, cmd, "clearing checked revisions should not emit a follow-up command")
	assert.Empty(t, ctx.CheckedItems, "esc should clear checked revisions in normal revisions mode")
}

func Test_UpdateStatus_UsesBindingDeclarationOrderForRevisions(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "revisions.move_down", Scope: "revisions", Key: config.StringList{"j"}},
		{Action: "revisions.move_up", Scope: "revisions", Key: config.StringList{"k"}},
		{Action: "revisions.open_rebase", Scope: "revisions", Key: config.StringList{"r"}},
		{Action: "ui.cancel", Scope: "ui", Key: config.StringList{"esc"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	ctx.Histories = config.NewHistories()
	model := NewUI(ctx)

	model.updateStatus()
	entries := help.FlatEntries(model.status.Help())
	assert.GreaterOrEqual(t, len(entries), 3)
	assert.Equal(t, "j", entries[0].Label)
	assert.Equal(t, "k", entries[1].Label)
	assert.Equal(t, "r", entries[2].Label)
}

func Test_UpdateStatus_IncludesAlwaysOnUiBindings(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "revisions.move_down", Scope: "revisions", Key: config.StringList{"j"}},
		{Action: "ui.show_command_history", Scope: "ui", Key: config.StringList{"W"}, Desc: "command history"},
	}

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.updateStatus()
	assert.Contains(t, help.FlatEntries(model.status.Help()), help.Entry{Label: "W", Desc: "command history"})
}

func Test_Update_SequencePrefixBeatsSingleKeyBinding(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.open_git", Scope: "revisions", Key: config.StringList{"g"}},
		{Action: "revset.edit", Scope: "revisions", Seq: config.StringList{"g", "r"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	// First key only starts pending sequence, should not trigger open_git.
	model.Update(tea.KeyPressMsg{Text: "g", Code: 'g'})
	assert.Nil(t, model.stacked)

	// Completing sequence should trigger ui.open_revset.
	model.Update(tea.KeyPressMsg{Text: "r", Code: 'r'})
	assert.True(t, model.revsetModel.Editing)
}

func Test_Update_PendingSequenceAutoExpandsStatusWithContinuations(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.open_git", Scope: "revisions", Key: config.StringList{"g"}},
		{Action: "revset.edit", Scope: "revisions", Seq: config.StringList{"g", "r"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_ = model.View()

	model.Update(tea.KeyPressMsg{Text: "g", Code: 'g'})
	assert.True(t, model.status.StatusExpanded(), "pending sequence should auto-expand status")

	model.updateStatus()
	entries := help.FlatEntries(model.status.Help())
	assert.NotEmpty(t, entries)
	assert.Equal(t, "r", entries[0].Label, "pending sequence should show continuation key")
}

func Test_Update_PendingSequenceMismatchClearsAutoExpandedStatus(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.open_git", Scope: "revisions", Key: config.StringList{"g"}},
		{Action: "revset.edit", Scope: "revisions", Seq: config.StringList{"g", "r"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_ = model.View()

	model.Update(tea.KeyPressMsg{Text: "g", Code: 'g'})
	assert.True(t, model.status.StatusExpanded())

	model.Update(tea.KeyPressMsg{Text: "x", Code: 'x'})
	assert.False(t, model.status.StatusExpanded(), "mismatched sequence should clear auto-expanded status")
}

func Test_Update_RevsetEditingInterceptsQuitKey(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "revset.edit", Scope: "revisions", Key: config.StringList{"L"}},
		{Action: "ui.quit", Scope: "ui", Key: config.StringList{"q"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.Update(tea.KeyPressMsg{Text: "L", Code: 'L'})
	assert.True(t, model.revsetModel.Editing)

	cmd := model.Update(tea.KeyPressMsg{Text: "q", Code: 'q'})
	assert.True(t, model.revsetModel.Editing, "q should be treated as text input while editing revset")
	if cmd != nil {
		msg := cmd()
		_, quit := msg.(tea.QuitMsg)
		assert.False(t, quit, "q in revset editing should not dispatch global quit")
	}
}

func Test_Update_GitFilterEditingEnterDoesNotTriggerApply(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "git.filter", Scope: "git", Key: config.StringList{"/"}},
		{Action: "git.apply", Scope: "git", Key: config.StringList{"enter"}},
		{Action: "ui.cancel", Scope: "ui", Key: config.StringList{"esc"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte(""))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)
	gitModel := git.NewModel(ctx, jj.NewSelectedRevisions())
	test.SimulateModel(gitModel, gitModel.Init())
	model.stacked = gitModel

	// Start filter editing.
	model.Update(tea.KeyPressMsg{Text: "/", Code: '/'})

	// Enter while editing applies filter only and must not execute actionApply.
	model.Update(tea.KeyPressMsg{Text: "f", Code: 'f'})
	model.Update(tea.KeyPressMsg{Text: "e", Code: 'e'})
	model.Update(tea.KeyPressMsg{Text: "t", Code: 't'})
	model.Update(tea.KeyPressMsg{Text: "c", Code: 'c'})
	model.Update(tea.KeyPressMsg{Text: "h", Code: 'h'})
	cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd, "enter in filter-edit mode should not dispatch apply")

	// Apply should now route through normal git scope after leaving filter-edit mode.
	_, handled := dispatchAction(model, keybindings.Action("git.apply"), nil)
	assert.True(t, handled, "apply should dispatch after filter-edit mode")
}

type scopeOnlyStackedModel struct {
	scope   string
	lastMsg tea.Msg
}

func (m *scopeOnlyStackedModel) Init() tea.Cmd {
	return nil
}

func (m *scopeOnlyStackedModel) Update(msg tea.Msg) tea.Cmd {
	m.lastMsg = msg
	return nil
}

func (m *scopeOnlyStackedModel) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (m *scopeOnlyStackedModel) Scopes() []dispatch.Scope {
	return []dispatch.Scope{
		{
			Name:    keybindings.ScopeName(m.scope),
			Leak:    dispatch.LeakNone,
			Handler: m,
		},
	}
}

func (m *scopeOnlyStackedModel) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	m.lastMsg = intent
	return nil, true
}

func Test_DispatchScopes_UsesStackedScope(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.stacked = &scopeOnlyStackedModel{scope: actions.ScopeUndo}
	scopes := model.dispatchScopes()
	require.NotEmpty(t, scopes)
	assert.Equal(t, keybindings.ScopeName(actions.ScopeUndo), scopes[0].Name)
}

func Test_HandleDispatchedAction_UsesStackedScope(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	stacked := &scopeOnlyStackedModel{scope: actions.ScopeChoose}
	model.stacked = stacked

	cmd, handled := dispatchAction(model, keybindings.Action("choose.move_down"), nil)
	assert.True(t, handled)
	assert.Nil(t, cmd)

	intent, ok := stacked.lastMsg.(intents.ChooseNavigate)
	assert.True(t, ok, "stacked model should receive choose intent via scope-based dispatch")
	if ok {
		assert.Equal(t, 1, intent.Delta)
	}
}

func Test_Update_BlockingScopeHandledNilCmdDoesNotReceiveRawKeyAgain(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "choose.cancel", Scope: "choose", Key: config.StringList{"esc"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	stacked := &scopeOnlyStackedModel{scope: actions.ScopeChoose}
	model.stacked = stacked

	cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.Nil(t, cmd, "choose.cancel handler returns nil cmd")

	_, ok := stacked.lastMsg.(intents.ChooseCancel)
	assert.True(t, ok, "blocking scope should keep the handled intent instead of receiving the raw key")
}

func Test_HandleDispatchedAction_RevisionsScopedActionInRebaseMode(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := rebase.NewOperation(
		ctx,
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "abc123", CommitId: "def456"}),
		rebase.SourceRevision,
		intents.ModeTargetDestination,
	)
	model.Update(common.RestoreOperationMsg{Operation: op})
	assert.False(t, model.revisions.InNormalMode(), "model should be in rebase mode")

	_, handled := dispatchAction(model, "revisions.move_down", nil)
	assert.True(t, handled, "revisions navigation actions should remain handled in rebase scope")
}

func Test_HandleDispatchedAction_RevisionsScopedActionInSetParentsMode(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetParents("abc123")).SetOutput([]byte("parent1"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := set_parents.NewModel(ctx, &jj.Commit{ChangeId: "abc123", CommitId: "def456"})
	model.Update(common.RestoreOperationMsg{Operation: op})
	assert.False(t, model.revisions.InNormalMode(), "model should be in set parents mode")

	_, handled := dispatchAction(model, "revisions.move_down", nil)
	assert.True(t, handled, "revisions navigation actions should remain handled in set parents scope")
}

func Test_HandleIntent_EditEntersRevsetInNormalMode(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	cmd, handled := model.HandleIntent(intents.Edit{})
	assert.True(t, handled)
	assert.NotNil(t, cmd)
	assert.True(t, model.revsetModel.Editing)
}

func Test_Update_RevsetScopedConfiguredActionDispatchesWhileEditing(t *testing.T) {
	origBindings := config.Current.Bindings
	origActions := config.Current.Actions
	defer func() {
		config.Current.Bindings = origBindings
		config.Current.Actions = origActions
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "revset.edit", Scope: "revisions", Key: config.StringList{"L"}},
		{Action: "revset_main_apply", Scope: "revset", Key: config.StringList{"ctrl+t"}},
	}
	config.Current.Actions = []config.ActionConfig{
		{Name: "revset_main_apply", Lua: `revset.set("main")`},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.Update(tea.KeyPressMsg{Text: "L", Code: 'L'})
	assert.True(t, model.revsetModel.Editing)

	cmd := model.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	assert.NotNil(t, cmd, "ctrl+t should dispatch revset-scoped custom action")
	if cmd != nil {
		msg := cmd()
		runLua, ok := msg.(common.RunLuaScriptMsg)
		assert.True(t, ok, "expected RunLuaScriptMsg from custom revset action")
		if ok {
			assert.Contains(t, runLua.Script, `revset.set("main")`)
		}
	}
}

func Test_Update_LuaActionDispatchesBuiltInAction(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	require.NoError(t, scripting.InitVM(ctx))
	defer scripting.CloseVM(ctx)
	model := NewUI(ctx)

	cmd := model.Update(common.RunLuaScriptMsg{Script: `jjui.revset.edit()`})
	assert.NotNil(t, cmd)

	test.SimulateModel(model, cmd)
	assert.True(t, model.revsetModel.Editing, "lua-dispatched revset.edit should enter revset editing")
}

func Test_Update_LuaRevsetSetWorksOutsideRevsetScope(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.CurrentRevset = "old"
	model := NewUI(ctx)

	cmd := model.Update(common.DispatchActionMsg{
		Action: "revset.set",
		Args:   map[string]any{"value": "new"},
	})
	require.NotNil(t, cmd)

	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)

	applied := false
	for _, batchCmd := range batch {
		msg := batchCmd()
		if revsetMsg, ok := msg.(common.UpdateRevSetMsg); ok {
			applied = true
			model.Update(revsetMsg)
		}
	}

	require.True(t, applied, "revset.set should emit an UpdateRevSetMsg")
	assert.Equal(t, "new", ctx.CurrentRevset)
}

func Test_Update_LuaBuiltinActionBypassesConfiguredOverride(t *testing.T) {
	origActions := config.Current.Actions
	defer func() {
		config.Current.Actions = origActions
	}()
	config.Current.Actions = []config.ActionConfig{
		{Name: "revset.edit", Lua: `flash("override")`},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	require.NoError(t, scripting.InitVM(ctx))
	defer scripting.CloseVM(ctx)
	model := NewUI(ctx)

	cmd := model.Update(common.RunLuaScriptMsg{Script: `jjui.revset.edit()`})
	require.NotNil(t, cmd)
	test.SimulateModel(model, cmd)
	assert.False(t, model.revsetModel.Editing, "override should replace default action behavior")

	cmd = model.Update(common.RunLuaScriptMsg{Script: `jjui.builtin.revset.edit()`})
	require.NotNil(t, cmd)
	test.SimulateModel(model, cmd)
	assert.True(t, model.revsetModel.Editing, "builtin action should bypass override and run default behavior")
}

func Test_Update_OperationScopedConfiguredActionOverridesBuiltInIntent(t *testing.T) {
	origActions := config.Current.Actions
	defer func() {
		config.Current.Actions = origActions
	}()
	config.Current.Actions = []config.ActionConfig{
		{Name: "revisions.details.diff", Lua: `flash("override")`},
	}

	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := details.NewOperation(ctx, &jj.Commit{ChangeId: "abc123", CommitId: "def456"})
	model.Update(common.RestoreOperationMsg{Operation: op})
	require.False(t, model.revisions.InNormalMode(), "details operation should be active")

	cmd := model.Update(common.DispatchActionMsg{Action: "revisions.details.diff"})
	require.NotNil(t, cmd)
	msg := cmd()
	runLua, ok := msg.(common.RunLuaScriptMsg)
	require.True(t, ok, "configured action should run before operation intent resolution")
	assert.Contains(t, runLua.Script, `flash("override")`)
}

func Test_Update_DispatchedDiffShowOpensAndUpdatesDiff(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	cmd := model.Update(common.DispatchActionMsg{
		Action:  "diff.show",
		Args:    map[string]any{"content": "new"},
		BuiltIn: true,
	})
	require.Nil(t, cmd)
	require.NotNil(t, model.diff)
	assert.Equal(t, "new", test.Stripped(test.RenderImmediate(model.diff, 20, 3)))
}

func Test_Update_DispatchedDiffShowUpdatesExistingDiff(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)
	model.diff = diff.New("old")

	cmd := model.Update(common.DispatchActionMsg{
		Action:  "diff.show",
		Args:    map[string]any{"content": "new"},
		BuiltIn: true,
	})
	require.Nil(t, cmd)
	require.NotNil(t, model.diff)
	assert.Equal(t, "new", test.Stripped(test.RenderImmediate(model.diff, 20, 3)))
}

func Test_Update_DiffEscClosesDiffAndRestoresDetails(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := details.NewOperation(ctx, &jj.Commit{ChangeId: "abc123", CommitId: "def456"})
	model.Update(common.RestoreOperationMsg{Operation: op})
	require.False(t, model.revisions.InNormalMode(), "details operation should be active")
	require.Equal(t, "details", model.revisions.CurrentOperation().Name())

	model.Update(intents.DiffShow{Content: "diff content"})
	require.NotNil(t, model.diff, "diff should open over details")

	cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	require.NotNil(t, cmd, "esc in diff should close diff")
	closeMsg, ok := cmd().(common.CloseViewMsg)
	require.True(t, ok, "esc in diff should dispatch close-view")

	model.Update(closeMsg)
	assert.Nil(t, model.diff, "diff should close after esc")
	require.False(t, model.revisions.InNormalMode(), "details should remain active after closing diff")
	assert.Equal(t, "details", model.revisions.CurrentOperation().Name())
}

func Test_Update_DispatchedPreviewShowUpdatesVisiblePreview(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)
	model.previewModel.SetVisible(true)
	model.previewModel.SetContent("old")

	cmd := model.Update(common.DispatchActionMsg{
		Action:  "ui.preview.show",
		Args:    map[string]any{"content": "new"},
		BuiltIn: true,
	})
	require.Nil(t, cmd)
	assert.Equal(t, "new", test.Stripped(test.RenderImmediate(model.previewModel, 20, 3)))
}

func Test_Update_DispatchedPreviewShowOpensHiddenPreview(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)
	model.previewModel.SetContent("old")
	model.previewModel.SetVisible(false)

	cmd := model.Update(common.DispatchActionMsg{
		Action:  "ui.preview.show",
		Args:    map[string]any{"content": "new"},
		BuiltIn: true,
	})
	require.Nil(t, cmd)
	assert.True(t, model.previewModel.Visible())
	assert.Equal(t, "new", test.Stripped(test.RenderImmediate(model.previewModel, 20, 3)))
}

func Test_Update_LuaInputEscCancelsAndFinishesScript(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "input.cancel", Scope: "input", Key: config.StringList{"esc"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	require.NoError(t, scripting.InitVM(ctx))
	defer scripting.CloseVM(ctx)
	model := NewUI(ctx)

	cmd := model.Update(common.RunLuaScriptMsg{Script: `local name = input("name")`})
	require.NotNil(t, cmd)
	test.SimulateModel(model, cmd)
	require.NotNil(t, model.scriptRunner, "script should wait for input")
	require.NotNil(t, model.stacked, "input view should be stacked")

	cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	require.NotNil(t, cmd, "esc in input scope should forward cancel to input model")
	test.SimulateModel(model, cmd)

	assert.Nil(t, model.stacked, "input should close after esc")
	assert.Nil(t, model.scriptRunner, "script should finish after input cancel")
}

func Test_Update_LuaChooseEscViaUiCancelFinishesScript(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.cancel", Scope: "ui", Key: config.StringList{"esc"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	require.NoError(t, scripting.InitVM(ctx))
	defer scripting.CloseVM(ctx)
	model := NewUI(ctx)

	cmd := model.Update(common.RunLuaScriptMsg{Script: `local choice = choose({"a", "b"})`})
	require.NotNil(t, cmd)
	test.SimulateModel(model, cmd)
	require.NotNil(t, model.scriptRunner, "script should wait for choose")
	require.NotNil(t, model.stacked, "choose view should be stacked")

	cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	require.NotNil(t, cmd, "esc should dispatch ui.cancel when choose.cancel is not configured")
	test.SimulateModel(model, cmd)

	assert.Nil(t, model.stacked, "choose should close after esc")
	assert.Nil(t, model.scriptRunner, "script should finish after choose cancel")
}

func Test_Update_LuaActionRejectsInvalidBuiltInArgs(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	require.NoError(t, scripting.InitVM(ctx))
	defer scripting.CloseVM(ctx)
	model := NewUI(ctx)

	cmd := model.Update(common.RunLuaScriptMsg{Script: `jjui.revert.set_target({ target = "bad" })`})
	assert.NotNil(t, cmd)

	test.SimulateModel(model, cmd)
	assert.True(t, model.flash.Any(), "invalid canonical action args should surface an error flash message")
}

func Test_Update_ExecHistoryUpDownNavigationInStatusInputScope(t *testing.T) {
	origBindings := config.Current.Bindings
	origSuggest := config.Current.Suggest.Exec.Mode
	defer func() {
		config.Current.Bindings = origBindings
		config.Current.Suggest.Exec.Mode = origSuggest
	}()

	config.Current.Suggest.Exec.Mode = "off"
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.exec_shell", Scope: "ui", Key: config.StringList{"$"}},
		{Action: "status.input.cancel", Scope: "status.input", Key: config.StringList{"esc"}},
		{Action: "status.input.apply", Scope: "status.input", Key: config.StringList{"enter"}},
		{Action: "status.input.autocomplete", Scope: "status.input", Key: config.StringList{"ctrl+r"}},
		{Action: "status.input.move_up", Scope: "status.input", Key: config.StringList{"up", "ctrl+p"}},
		{Action: "status.input.move_down", Scope: "status.input", Key: config.StringList{"down", "ctrl+n"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.Histories = config.NewHistories()
	history := ctx.Histories.GetHistory(config.HistoryKey("exec sh"), true)
	history.Append("first-cmd")
	history.Append("second-cmd")

	model := NewUI(ctx)

	model.Update(tea.KeyPressMsg{Text: "$", Code: '$'})
	assert.True(t, model.status.IsFocused(), "exec shell should focus status input")

	model.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	firstNav := model.status.InputValue()
	assert.NotEmpty(t, firstNav, "up should navigate to a history command")

	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	secondNav := model.status.InputValue()
	assert.NotEmpty(t, secondNav, "down should navigate to a history command")
	assert.NotEqual(t, firstNav, secondNav, "down should move to a different history entry")
}

func Test_UpdateStatus_RevsetEditingUsesDispatcherHelpWhenAvailable(t *testing.T) {
	origBindings := config.Current.Bindings
	origActions := config.Current.Actions
	defer func() {
		config.Current.Bindings = origBindings
		config.Current.Actions = origActions
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "revset_main_apply", Scope: "revset", Key: config.StringList{"ctrl+t"}},
	}
	config.Current.Actions = []config.ActionConfig{
		{Name: "revset_main_apply", Lua: `revset.set("main")`},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.revsetModel.Update(revset.EditRevSetMsg{})
	assert.True(t, model.revsetModel.Editing)

	model.updateStatus()
	entries := help.FlatEntries(model.status.Help())
	assert.NotEmpty(t, entries)
	assert.Equal(t, "ctrl+t", entries[0].Label)
}

func Test_UpdateStatus_CustomLuaActionUsesConfiguredDescription(t *testing.T) {
	origBindings := config.Current.Bindings
	origActions := config.Current.Actions
	defer func() {
		config.Current.Bindings = origBindings
		config.Current.Actions = origActions
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "my_quit", Desc: "My quit", Scope: "revisions", Key: config.StringList{"x"}},
	}
	config.Current.Actions = []config.ActionConfig{
		{Name: "my_quit", Lua: `print("quit")`},
	}

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.updateStatus()
	entries := help.FlatEntries(model.status.Help())
	assert.NotEmpty(t, entries)
	assert.Equal(t, "My quit", entries[0].Desc)
}

func Test_Update_InlineDescribeDispatcherKeysWorkWhileEditing(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "revisions.inline_describe.cancel", Scope: "revisions.inline_describe", Key: config.StringList{"esc"}},
		{Action: "revisions.inline_describe.accept", Scope: "revisions.inline_describe", Key: config.StringList{"alt+enter"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("abc123")).SetOutput([]byte("old desc"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := describe.NewOperation(ctx, &jj.Commit{ChangeId: "abc123", CommitId: "def456"})
	model.Update(common.RestoreOperationMsg{Operation: op})
	scopes := model.dispatchScopes()
	require.NotEmpty(t, scopes)
	require.Equal(t, keybindings.ScopeName(actions.ScopeInlineDescribe), scopes[0].Name)
	foundCancel := false
	foundAccept := false
	for _, b := range config.BindingsToRuntime(config.Current.Bindings) {
		if b.Scope != keybindings.ScopeName(actions.ScopeInlineDescribe) {
			continue
		}
		if b.Action == "revisions.inline_describe.cancel" {
			foundCancel = true
		}
		if b.Action == "revisions.inline_describe.accept" {
			foundAccept = true
		}
	}
	require.True(t, foundCancel)
	require.True(t, foundAccept)
	cmd, handled := dispatchAction(model, "revisions.inline_describe.cancel", nil)
	require.True(t, handled)
	require.NotNil(t, cmd)

	// esc should dispatch cancel intent for inline describe.
	cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.NotNil(t, cmd)
	if cmd != nil {
		_, ok := cmd().(common.CloseViewMsg)
		assert.True(t, ok, "esc should close inline describe via dispatcher")
	}

	// Verify alt+enter dispatches inline_describe_accept while editing.
	cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt})
	assert.NotNil(t, cmd, "alt+enter should trigger inline_describe_accept via dispatcher")
}

func Test_Update_DetailsCloseClearsSelectedFiles(t *testing.T) {
	const statusOutput = "false false $\nM file.txt\nA newfile.txt\n"

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status("abc123")).SetOutput([]byte(statusOutput))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := details.NewOperation(ctx, &jj.Commit{ChangeId: "abc123", CommitId: "def456"})
	model.Update(common.RestoreOperationMsg{Operation: op})
	test.SimulateModel(model, op.Init())
	require.False(t, model.revisions.InNormalMode(), "details operation should be active")

	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	require.Len(t, ctx.CheckedItems, 1, "details selection should be tracked before close")

	test.SimulateModel(model, test.Press(tea.KeyEsc))
	assert.True(t, model.revisions.InNormalMode(), "esc should close details")
	assert.Empty(t, ctx.CheckedItems, "closing details should clear selected files from context")
}

func Test_Update_RestoreDetailsOperationResyncsSelectedFiles(t *testing.T) {
	const statusOutput = "false false $\nM file.txt\nA newfile.txt\n"

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status("abc123")).SetOutput([]byte(statusOutput))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := details.NewOperation(ctx, &jj.Commit{ChangeId: "abc123", CommitId: "def456"})
	model.Update(common.RestoreOperationMsg{Operation: op})
	test.SimulateModel(model, op.Init())

	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	require.Len(t, ctx.CheckedItems, 1, "details selection should be tracked before close")

	model.Update(common.CloseViewMsg{})
	assert.Empty(t, ctx.CheckedItems, "closing details should clear selected files from context")

	model.Update(common.RestoreOperationMsg{Operation: op})
	assert.Len(t, ctx.CheckedItems, 1, "restoring details should resync checked files from the operation state")
}

func Test_Update_DetailsEscClosesOperation(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "revisions.details.cancel", Scope: "revisions.details", Key: config.StringList{"esc"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := details.NewOperation(ctx, &jj.Commit{ChangeId: "abc123", CommitId: "def456"})
	model.Update(common.RestoreOperationMsg{Operation: op})
	require.False(t, model.revisions.InNormalMode(), "details operation should be active")

	cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	require.NotNil(t, cmd, "esc should resolve to revisions.details.cancel")

	msg := cmd()
	closeMsg, ok := msg.(common.CloseViewMsg)
	require.True(t, ok, "esc should dispatch a close-view message from details")
	assert.False(t, closeMsg.Applied, "plain esc should close without applied state")

	model.Update(closeMsg)
	assert.True(t, model.revisions.InNormalMode(), "details esc should close details operation")
}

func Test_Update_DetailsEscClosesOperation_WithDefaultBindings(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := details.NewOperation(ctx, &jj.Commit{ChangeId: "abc123", CommitId: "def456"})
	model.Update(common.RestoreOperationMsg{Operation: op})
	require.False(t, model.revisions.InNormalMode(), "details operation should be active")

	cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	require.NotNil(t, cmd, "default esc binding should resolve in details scope")

	msg := cmd()
	closeMsg, ok := msg.(common.CloseViewMsg)
	require.True(t, ok, "default details esc should dispatch a close-view message")

	model.Update(closeMsg)
	assert.True(t, model.revisions.InNormalMode(), "default details esc should close details operation")
}

func Test_Update_CommandErrorAfterClosingDetailsWithSelectedFiles_AllowsEscToDismissFlash(t *testing.T) {
	const statusOutput = "false false $\nM file.txt\nA newfile.txt\n"

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status("abc123")).SetOutput([]byte(statusOutput))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := details.NewOperation(ctx, &jj.Commit{ChangeId: "abc123", CommitId: "def456"})
	model.Update(common.RestoreOperationMsg{Operation: op})
	test.SimulateModel(model, op.Init())
	require.False(t, model.revisions.InNormalMode(), "details operation should be active")

	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	require.Len(t, ctx.CheckedItems, 1, "details selection should be tracked before close")

	model.Update(common.CloseViewMsg{})
	model.Update(common.CommandCompletedMsg{Err: errors.New("split failed")})

	assert.True(t, model.revisions.InNormalMode(), "closing details should return to revisions")
	assert.True(t, model.flash.Any(), "command failure should surface as a flash message")
	assert.Empty(t, ctx.CheckedItems, "selected files should be cleared when leaving details")

	cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.Nil(t, cmd, "esc should dismiss the flash instead of being consumed by stale checked items")
	assert.False(t, model.flash.Any(), "esc should dismiss the split error flash")
}

func Test_Update_SetBookmarkTypingDoesNotTogglePreview(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.preview_toggle", Scope: "ui", Key: config.StringList{"p"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovable("abc123")).SetOutput([]byte(""))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)
	model.previewModel.SetVisible(true)

	op := bookmark.NewSetBookmarkOperation(ctx, "abc123", "")
	test.SimulateModel(op, op.Init())
	model.Update(common.RestoreOperationMsg{Operation: op})
	require.False(t, model.revisions.InNormalMode(), "set bookmark operation should be active")
	require.True(t, model.revisions.IsEditing(), "set bookmark should be editing")

	test.SimulateModel(model, test.Type("p"))
	assert.True(t, model.previewModel.Visible(), "typing in set_bookmark should not toggle preview")
}

// withShortColorSchemePoll temporarily shortens the polling interval so
// tests don't have to wait the full default duration for tea.Tick to fire.
func withShortColorSchemePoll(t *testing.T) {
	t.Helper()
	orig := colorSchemePollInterval
	colorSchemePollInterval = time.Millisecond
	t.Cleanup(func() { colorSchemePollInterval = orig })
}

// drainCmds expands a Cmd tree, invoking each leaf Cmd and feeding the
// resulting tea.BatchMsg back through the queue. visit is called once for
// every non-batch leaf Cmd (with the Cmd itself and the message it produced,
// which may be nil). Stop draining by returning false.
func drainCmds(root tea.Cmd, visit func(c tea.Cmd, msg tea.Msg) bool) {
	queue := []tea.Cmd{root}
	for len(queue) > 0 {
		var c tea.Cmd
		c, queue = queue[0], queue[1:]
		if c == nil {
			continue
		}
		msg := c()
		if batch, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, batch...)
			continue
		}
		if !visit(c, msg) {
			return
		}
	}
}

func Test_Init_EnablesMode2031AndStartsPolling(t *testing.T) {
	withShortColorSchemePoll(t)

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	w := New(ctx)

	cmd := w.Init()
	require.NotNil(t, cmd)

	var foundEnable2031, foundProbe2031, foundPollTick bool

	drainCmds(cmd, func(c tea.Cmd, msg tea.Msg) bool {
		switch v := msg.(type) {
		case tea.RawMsg:
			switch v.Msg {
			case ansi.SetModeLightDark:
				foundEnable2031 = true
			case ansi.RequestModeLightDark:
				foundProbe2031 = true
			}
		case colorSchemePollTickMsg:
			foundPollTick = true
		}
		return true
	})

	assert.True(t, foundEnable2031)
	assert.True(t, foundProbe2031)
	assert.True(t, foundPollTick)
}

func Test_PollTick_RequestsBackgroundColorAndRearms(t *testing.T) {
	withShortColorSchemePoll(t)

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	cmd := model.Update(colorSchemePollTickMsg{})
	require.NotNil(t, cmd)

	wantRequestPC := reflect.ValueOf(tea.RequestBackgroundColor).Pointer()

	var foundRequest, foundRearm bool
	drainCmds(cmd, func(c tea.Cmd, msg tea.Msg) bool {
		if reflect.ValueOf(c).Pointer() == wantRequestPC {
			foundRequest = true
			return true
		}
		if _, ok := msg.(colorSchemePollTickMsg); ok {
			foundRearm = true
		}
		return true
	})
	assert.True(t, foundRequest)
	assert.True(t, foundRearm)
}

func Test_PollTick_StopsWhenMode2031Supported(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.Update(tea.ModeReportMsg{Mode: ansi.ModeLightDark, Value: ansi.ModeSet})
	assert.True(t, model.mode2031Supported)

	cmd := model.Update(colorSchemePollTickMsg{})
	assert.Nil(t, cmd)
}

func Test_PollTick_ContinuesWhenMode2031NotRecognized(t *testing.T) {
	withShortColorSchemePoll(t)

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.Update(tea.ModeReportMsg{Mode: ansi.ModeLightDark, Value: ansi.ModeNotRecognized})
	assert.False(t, model.mode2031Supported)

	cmd := model.Update(colorSchemePollTickMsg{})
	assert.NotNil(t, cmd, "polling should continue when mode 2031 is not recognized")
}

func Test_ResumeMsg_ReEnablesMode2031AndQueriesBackground(t *testing.T) {
	withShortColorSchemePoll(t)

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	cmd := model.Update(tea.ResumeMsg{})
	require.NotNil(t, cmd)

	wantRequestPC := reflect.ValueOf(tea.RequestBackgroundColor).Pointer()

	var foundEnable2031, foundProbe2031, foundBgRequest, foundPollTick bool
	drainCmds(cmd, func(c tea.Cmd, msg tea.Msg) bool {
		if reflect.ValueOf(c).Pointer() == wantRequestPC {
			foundBgRequest = true
			return true
		}
		switch v := msg.(type) {
		case tea.RawMsg:
			switch v.Msg {
			case ansi.SetModeLightDark:
				foundEnable2031 = true
			case ansi.RequestModeLightDark:
				foundProbe2031 = true
			}
		case colorSchemePollTickMsg:
			foundPollTick = true
		}
		return true
	})

	assert.True(t, foundEnable2031, "resume should re-enable mode 2031")
	assert.True(t, foundBgRequest, "resume should re-query background color")
	assert.False(t, foundProbe2031, "resume should not re-probe mode 2031 support; the initial probe result still applies")
	assert.False(t, foundPollTick, "resume should not restart polling; the existing poll loop survives suspension")
}
