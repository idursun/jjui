package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/git"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/describe"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/operations/rebase"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Update_RevsetWithEmptyInputKeepsDefaultRevset(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.DefaultRevset = "assume-passed-from-cli"

	model := NewUI(ctx)
	model.Update(common.UpdateRevSetMsg(""))

	assert.Equal(t, ctx.DefaultRevset, ctx.CurrentRevset)
}

func Test_Update_PreviewScrollKeysWorkWhenVisible(t *testing.T) {
	tests := []struct {
		name           string
		key            tea.KeyMsg
		expectedScroll int // positive = down, negative = up
	}{
		{
			name:           "ctrl+d scrolls half page down",
			key:            tea.KeyMsg{Type: tea.KeyCtrlD},
			expectedScroll: 1,
		},
		{
			name:           "ctrl+u scrolls half page up",
			key:            tea.KeyMsg{Type: tea.KeyCtrlU},
			expectedScroll: -1,
		},
		{
			name:           "ctrl+n scrolls down",
			key:            tea.KeyMsg{Type: tea.KeyCtrlN},
			expectedScroll: 1,
		},
		{
			name:           "ctrl+p scrolls up",
			key:            tea.KeyMsg{Type: tea.KeyCtrlP},
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
			model.previewModel.ViewRect(render.NewDisplayContext(), layout.NewBox(cellbuf.Rect(0, 0, 100, 50)))

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
		key            tea.KeyMsg
		expectedResize int // positive = expand, negative = shrink
	}{
		{
			name:           "ctrl+l shrinks preview",
			key:            tea.KeyMsg{Type: tea.KeyCtrlL},
			expectedResize: -1,
		},
		{
			name:           "ctrl+h expands preview",
			key:            tea.KeyMsg{Type: tea.KeyCtrlH},
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

	// Verify status has higher z-index than git
	dl := render.NewDisplayContext()
	box := layout.NewBox(cellbuf.Rect(0, 0, 100, 40))
	model.stacked.ViewRect(dl, box)
	gitDraws := dl.DrawList()
	assert.NotEmpty(t, gitDraws, "git should produce draw operations")

	maxGitZ := 0
	for _, draw := range gitDraws {
		if draw.Z > maxGitZ {
			maxGitZ = draw.Z
		}
	}
	assert.Less(t, maxGitZ, render.ZExpandedStatus,
		"git z-index (%d) should be less than ZExpandedStatus (%d)", maxGitZ, render.ZExpandedStatus)

	// Press 'esc' to close stacked first.
	test.SimulateModel(model, test.Press(tea.KeyEscape))
	assert.True(t, model.status.StatusExpanded(), "status should remain expanded while stacked is closed first")

	// Stacked (git) should be closed first
	assert.Nil(t, model.stacked, "stacked (git) should close before expanded status")
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

	model.state = common.Error
	model.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	assert.Equal(t, common.Ready, model.state, "ctrl+x should use configured global cancel binding")

	model.state = common.Error
	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, common.Error, model.state, "esc should not act as global cancel when global bindings are configured")
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
	entries := model.status.Help()
	assert.GreaterOrEqual(t, len(entries), 3)
	assert.Equal(t, "j", entries[0].Label)
	assert.Equal(t, "k", entries[1].Label)
	assert.Equal(t, "r", entries[2].Label)
}

func Test_Update_SequencePrefixBeatsSingleKeyBinding(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.open_git", Scope: "revisions", Key: config.StringList{"g"}},
		{Action: "ui.open_revset", Scope: "revisions", Seq: config.StringList{"g", "r"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	// First key only starts pending sequence, should not trigger open_git.
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	assert.Nil(t, model.stacked)

	// Completing sequence should trigger ui.open_revset.
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	assert.True(t, model.revsetModel.Editing)
}

func Test_Update_PendingSequenceAutoExpandsStatusWithContinuations(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.open_git", Scope: "revisions", Key: config.StringList{"g"}},
		{Action: "ui.open_revset", Scope: "revisions", Seq: config.StringList{"g", "r"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_ = model.View()

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	assert.True(t, model.status.StatusExpanded(), "pending sequence should auto-expand status")

	model.updateStatus()
	entries := model.status.Help()
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
		{Action: "ui.open_revset", Scope: "revisions", Seq: config.StringList{"g", "r"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	_ = model.View()

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	assert.True(t, model.status.StatusExpanded())

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	assert.False(t, model.status.StatusExpanded(), "mismatched sequence should clear auto-expanded status")
}

func Test_Update_RevsetEditingInterceptsQuitKey(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.open_revset", Scope: "revisions", Key: config.StringList{"L"}},
		{Action: "ui.quit", Scope: "ui", Key: config.StringList{"q"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	assert.True(t, model.revsetModel.Editing)

	cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	assert.True(t, model.revsetModel.Editing, "q should be treated as text input while editing revset")
	if cmd != nil {
		msg := cmd()
		_, quit := msg.(tea.QuitMsg)
		assert.False(t, quit, "q in revset editing should not dispatch global quit")
	}
}

func Test_Update_GitUnmatchedShortcutFallback(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "git.filter", Scope: "git", Key: config.StringList{"/"}},
		{Action: "git.apply", Scope: "git.filter", Key: config.StringList{"enter"}},
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
	test.SimulateModel(gitModel, func() tea.Msg { return intents.GitFilter{Kind: intents.GitFilterPush} })

	assert.NotNil(t, model.stacked)

	// 'p' is unmatched in dispatcher for this test and should be forwarded to git shortcut handling.
	cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	assert.NotNil(t, cmd, "unmatched git shortcut should be forwarded to git model fallback")
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
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})

	// Enter while editing applies filter only and must not execute actionApply.
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "enter in filter-edit mode should not dispatch apply")

	// Apply should now route through normal git scope after leaving filter-edit mode.
	_, handled := model.handleDispatchedAction(actions.GitApply, nil)
	assert.True(t, handled, "apply should dispatch after filter-edit mode")
}

type ownerOnlyStackedModel struct {
	owner   string
	lastMsg tea.Msg
}

func (m *ownerOnlyStackedModel) Init() tea.Cmd {
	return nil
}

func (m *ownerOnlyStackedModel) Update(msg tea.Msg) tea.Cmd {
	m.lastMsg = msg
	return nil
}

func (m *ownerOnlyStackedModel) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (m *ownerOnlyStackedModel) StackedActionOwner() string {
	return m.owner
}

func Test_ActiveScopeChain_UsesStackedOwnerScope(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.stacked = &ownerOnlyStackedModel{owner: actions.OwnerUndo}
	scopes := model.activeScopeChain()
	require.NotEmpty(t, scopes)
	assert.Equal(t, keybindings.Scope(actions.OwnerUndo), scopes[0])
}

func Test_HandleDispatchedAction_UsesStackedOwnerScope(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	stacked := &ownerOnlyStackedModel{owner: actions.OwnerChoose}
	model.stacked = stacked

	cmd, handled := model.handleDispatchedAction(actions.ChooseMoveDown, nil)
	assert.True(t, handled)
	assert.Nil(t, cmd)

	intent, ok := stacked.lastMsg.(intents.ChooseNavigate)
	assert.True(t, ok, "stacked model should receive choose intent via owner-based dispatch")
	if ok {
		assert.Equal(t, 1, intent.Delta)
	}
}

func Test_HandleDispatchedAction_RevisionsScopedActionInRebaseMode(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := rebase.NewOperation(
		ctx,
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "abc123", CommitId: "def456"}),
		rebase.SourceRevision,
		rebase.TargetDestination,
	)
	model.Update(common.RestoreOperationMsg{Operation: op})
	assert.False(t, model.revisions.InNormalMode(), "model should be in rebase mode")

	_, handled := model.handleDispatchedAction("revisions.move_down", nil)
	assert.True(t, handled, "revisions navigation actions should remain handled in rebase scope")
}

func Test_HandleDelegatedIntent_EditEntersRevsetInNormalMode(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	cmd, handled := model.handleDelegatedIntent(intents.Edit{Clear: true})
	assert.True(t, handled)
	assert.NotNil(t, cmd)
	assert.True(t, model.revsetModel.Editing)
}

func Test_HandleDelegatedIntent_EditIgnoredOutsideNormalMode(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := rebase.NewOperation(
		ctx,
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "abc123", CommitId: "def456"}),
		rebase.SourceRevision,
		rebase.TargetDestination,
	)
	model.Update(common.RestoreOperationMsg{Operation: op})
	assert.False(t, model.revisions.InNormalMode())

	cmd, handled := model.handleDelegatedIntent(intents.Edit{Clear: true})
	assert.True(t, handled)
	assert.Nil(t, cmd)
	assert.False(t, model.revsetModel.Editing)
}

func Test_Update_RevsetScopedConfiguredActionDispatchesWhileEditing(t *testing.T) {
	origBindings := config.Current.Bindings
	origActions := config.Current.Actions
	defer func() {
		config.Current.Bindings = origBindings
		config.Current.Actions = origActions
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.open_revset", Scope: "revisions", Key: config.StringList{"L"}},
		{Action: "revset_main_apply", Scope: "revset", Key: config.StringList{"ctrl+t"}},
	}
	config.Current.Actions = []config.ActionConfig{
		{Name: "revset_main_apply", Desc: "Set revset to main", Lua: `revset.set("main")`},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	assert.True(t, model.revsetModel.Editing)

	cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
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
	model := NewUI(ctx)

	cmd := model.Update(common.RunLuaScriptMsg{Script: `jjui.ui.open_revset()`})
	assert.NotNil(t, cmd)

	test.SimulateModel(model, cmd)
	assert.True(t, model.revsetModel.Editing, "lua-dispatched ui.open_revset should enter revset editing")
}

func Test_Update_LuaActionRejectsInvalidBuiltInArgs(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
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

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("$")})
	assert.True(t, model.status.IsFocused(), "exec shell should focus status input")

	model.Update(tea.KeyMsg{Type: tea.KeyUp})
	firstNav := model.status.InputValue()
	assert.NotEmpty(t, firstNav, "up should navigate to a history command")

	model.Update(tea.KeyMsg{Type: tea.KeyDown})
	secondNav := model.status.InputValue()
	assert.NotEmpty(t, secondNav, "down should navigate to a history command")
	assert.NotEqual(t, firstNav, secondNav, "down should move to a different history entry")
}

func Test_Update_RevsetEnterAppliesAndEscCancelsViaDispatcher(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.open_revset", Scope: "revisions", Key: config.StringList{"L"}},
		{Action: "revset.apply", Scope: "revset", Key: config.StringList{"enter"}},
		{Action: "revset.cancel", Scope: "revset", Key: config.StringList{"esc"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.DefaultRevset = "@"
	model := NewUI(ctx)

	// Enter should apply current text in revset editing.
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	assert.True(t, model.revsetModel.Editing)
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, cmd)
	assert.False(t, model.revsetModel.Editing)

	// Esc should cancel revset editing mode.
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	assert.True(t, model.revsetModel.Editing)
	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, model.revsetModel.Editing)
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
		{Name: "revset_main_apply", Desc: "Set revset to main", Lua: `revset.set("main")`},
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
	entries := model.status.Help()
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
		{Action: "my_quit", Scope: "revisions", Key: config.StringList{"x"}},
	}
	config.Current.Actions = []config.ActionConfig{
		{Name: "my_quit", Desc: "My quit", Lua: `print("quit")`},
	}

	commandRunner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.updateStatus()
	entries := model.status.Help()
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
	require.Equal(t, keybindings.Scope(actions.OwnerInlineDescribe), model.activeScopeChain()[0])
	foundCancel := false
	foundAccept := false
	for _, b := range config.BindingsToRuntime(config.Current.Bindings) {
		if b.Scope != keybindings.Scope(actions.OwnerInlineDescribe) {
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
	cmd, handled := model.handleDispatchedAction("revisions.inline_describe.cancel", nil)
	require.True(t, handled)
	require.NotNil(t, cmd)

	// esc should dispatch cancel intent for inline describe.
	cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)
	if cmd != nil {
		_, ok := cmd().(common.CloseViewMsg)
		assert.True(t, ok, "esc should close inline describe via dispatcher")
	}

	// Verify alt+enter dispatches inline_describe_accept while editing.
	cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	assert.NotNil(t, cmd, "alt+enter should trigger inline_describe_accept via dispatcher")
}

func Test_Update_TargetPickerEscCancelsEditing(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "revisions.rebase.target", Scope: "revisions.rebase", Key: config.StringList{"t"}},
		{Action: "revisions.target_picker.cancel", Scope: "revisions.target_picker", Key: config.StringList{"esc"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()
	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := rebase.NewOperation(
		ctx,
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "abc123", CommitId: "def456"}),
		rebase.SourceRevision,
		rebase.TargetDestination,
	)
	model.Update(common.RestoreOperationMsg{Operation: op})

	test.SimulateModel(model, test.Type("t"))
	assert.True(t, model.revisions.IsEditing(), "target picker should open on rebase_target action")

	test.SimulateModel(model, test.Press(tea.KeyEsc))
	assert.False(t, model.revisions.IsEditing(), "esc in target scope should cancel target picker editing")
}

func Test_Update_DetailsCancelPrecedenceOverFlashDismissal(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "revisions.details.cancel", Scope: "revisions.details", Key: config.StringList{"h"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	op := details.NewOperation(ctx, &jj.Commit{ChangeId: "abc123", CommitId: "def456"})
	model.Update(common.RestoreOperationMsg{Operation: op})
	require.False(t, model.revisions.InNormalMode(), "details operation should be active")

	model.Update(intents.AddMessage{Text: "flash", Sticky: true})
	require.True(t, model.flash.Any(), "flash should be visible before cancel")

	test.SimulateModel(model, test.Type("h"))
	assert.True(t, model.revisions.InNormalMode(), "details cancel should close details operation")
	assert.True(t, model.flash.Any(), "details cancel should not dismiss flash first")
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

	op := bookmark.NewSetBookmarkOperation(ctx, "abc123")
	test.SimulateModel(op, op.Init())
	model.Update(common.RestoreOperationMsg{Operation: op})
	require.False(t, model.revisions.InNormalMode(), "set bookmark operation should be active")
	require.True(t, model.revisions.IsEditing(), "set bookmark should be editing")

	test.SimulateModel(model, test.Type("p"))
	assert.True(t, model.previewModel.Visible(), "typing in set_bookmark should not toggle preview")
}

func Test_Update_QuickSearchEscClearsQuickSearchText(t *testing.T) {
	origBindings := config.Current.Bindings
	defer func() {
		config.Current.Bindings = origBindings
	}()
	config.Current.Bindings = []config.BindingConfig{
		{Action: "ui.quick_search", Scope: "ui", Key: config.StringList{"/"}},
		{Action: "revisions.quick_search_clear", Scope: "revisions.quick_search", Key: config.StringList{"esc"}},
	}

	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.Histories = config.NewHistories()
	model := NewUI(ctx)

	model.Update(common.QuickSearchMsg("second"))
	assert.True(t, model.revisions.HasQuickSearch(), "quick search should be active after setting search text")

	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, model.revisions.HasQuickSearch(), "esc in quick_search scope should clear quick search text")
}

func Test_Update_FileSearchTypingUpdatesStatusInput(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.Histories = config.NewHistories()
	model := NewUI(ctx)

	model.Update(common.FileSearchMsg{
		Revset:       "@",
		PreviewShown: false,
		Commit:       &jj.Commit{ChangeId: "abc123", CommitId: "def456"},
		RawFileOut:   []byte("a.txt\nb.txt"),
	})
	assert.True(t, model.status.IsFocused(), "file search should focus status input")

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	assert.Equal(t, "x", model.status.InputValue(), "typed key should update file-search input")
}
