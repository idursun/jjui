package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/bookmarkpane"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBookmarkPaneModel(t *testing.T, output string) *Model {
	t.Helper()
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(output))
	t.Cleanup(commandRunner.Verify)

	ctx := test.NewTestContext(commandRunner)
	return NewUI(ctx)
}

func bookmarkPaneFocused(model *Model) bool {
	scopes := model.dispatchScopes()
	return len(scopes) > 0 && (scopes[0].Name == actions.ScopeBookmarkPane ||
		scopes[0].Name == actions.ScopeBookmarkPaneConfirmation ||
		scopes[0].Name == actions.ScopeBookmarkPaneFilter)
}

func bookmarkPaneVisible(model *Model) bool {
	return strings.Contains(renderSplitView(model, 100, 20), "Bookmarks")
}

func Test_BookmarkPaneReadlineNavigation(t *testing.T) {
	model := newBookmarkPaneModel(t, "local;.;true;false;false;false;abc123\nremote;origin;true;true;false;false;def456\n")
	test.SimulateModel(model, model.Update(intents.ToggleBookmarkPane{}))
	require.True(t, bookmarkPaneFocused(model))

	for _, step := range []struct {
		key      rune
		hasLocal bool
	}{
		{'p', true},  // Clamp at the first bookmark.
		{'n', false}, // Move to the remote-only bookmark.
		{'n', false}, // Clamp at the last bookmark.
		{'p', true},  // Return to the local bookmark.
	} {
		test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: step.key, Mod: tea.ModCtrl}))
		value, ok := model.QueryState("bookmark_pane.has_local_bookmark")
		require.True(t, ok)
		assert.Equal(t, step.hasLocal, value, "after ctrl+%c", step.key)
	}
}

func Test_GitOverlayDoesNotHideBehindBookmarkPane(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin\n"))
	t.Cleanup(commandRunner.Verify)
	model := NewUI(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.Update(intents.ToggleBookmarkPane{}))
	test.SimulateModel(model, model.Update(intents.FocusNextPane{}))
	_, handled := model.HandleIntent(intents.OpenGit{})
	require.True(t, handled)
	require.Equal(t, bookmarkContentID, model.splitContainer.ActiveID(), "opening Git should preserve the bookmark pane")

	model.width, model.height = 100, 24
	view := ansi.Strip(model.View())
	lines := strings.Split(view, "\n")
	// At this width the Git modal spans the split boundary. Its right border
	// must be visible even where the bookmark pane fills the background.
	border := []rune(lines[2])
	assert.Equal(t, '│', border[88], "Git modal's right border was obscured:\n%s", view)
	assert.Contains(t, view, "Git Operations")
	assert.NotContains(t, lines[1], "Bookmarks", "bookmark title must stay behind Git's top border")

	// Closing Git reveals the still-open bookmark pane without reopening it.
	test.SimulateModel(model, func() tea.Msg { return common.CloseViewMsg{} })
	require.Equal(t, bookmarkContentID, model.splitContainer.ActiveID())
	assert.Contains(t, renderSplitView(model, 100, 20), "Bookmarks")
}

func Test_ToggleBookmarkPane_OpensFocusedPaneAndTabReturnsFocusToRevisions(t *testing.T) {
	model := newBookmarkPaneModel(t, "main;.;true;false;false;false;abc123\n")

	test.SimulateModel(model, model.Update(intents.ToggleBookmarkPane{}))
	assert.True(t, bookmarkPaneVisible(model))
	assert.True(t, bookmarkPaneFocused(model))
	assert.False(t, model.revisions.IsFocused())
	require.NotEmpty(t, model.dispatchScopes())
	assert.Equal(t, keybindings.ScopeName(actions.ScopeBookmarkPane), model.dispatchScopes()[0].Name)

	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: tea.KeyTab}))
	assert.False(t, bookmarkPaneFocused(model))
	assert.True(t, model.revisions.IsFocused())
	require.NotEmpty(t, model.dispatchScopes())
	assert.Equal(t, keybindings.ScopeName(actions.ScopeRevisions), model.dispatchScopes()[0].Name)

	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: tea.KeyTab}))
	assert.True(t, bookmarkPaneFocused(model))
	assert.False(t, model.revisions.IsFocused())

	test.SimulateModel(model, model.Update(intents.ToggleBookmarkPane{}))
	assert.False(t, bookmarkPaneVisible(model))
	assert.True(t, model.revisions.IsFocused())
}

func Test_BookmarkPaneShortcutsDoNotShadowRevisionsWhenRevisionsFocused(t *testing.T) {
	model := newBookmarkPaneModel(t, "main;.;true;false;false;false;abc123\n")

	test.SimulateModel(model, model.Update(intents.ToggleBookmarkPane{}))
	test.SimulateModel(model, model.Update(intents.FocusNextPane{}))
	require.False(t, bookmarkPaneFocused(model))
	require.True(t, model.revisions.IsFocused())

	result := model.resolver.ResolveKey(tea.KeyPressMsg{Text: "p", Code: 'p'}, model.dispatchScopes())
	_, ok := result.Intent.(intents.PreviewToggle)
	assert.True(t, ok, "revision shortcuts should win while revisions are focused")
}

func Test_BookmarkPaneCancel_ClosesPaneAndRestoresRevisionFocus(t *testing.T) {
	model := newBookmarkPaneModel(t, "main;.;true;false;false;false;abc123\n")

	test.SimulateModel(model, model.Update(intents.ToggleBookmarkPane{}))
	require.True(t, bookmarkPaneFocused(model))

	test.SimulateModel(model, model.Update(intents.Cancel{}))

	assert.False(t, bookmarkPaneVisible(model))
	assert.True(t, model.revisions.IsFocused())
}

func Test_CloseView_DoesNotCloseUnfocusedBookmarkPane(t *testing.T) {
	model := newBookmarkPaneModel(t, "main;.;true;false;false;false;abc123\n")

	test.SimulateModel(model, model.Update(intents.ToggleBookmarkPane{}))
	test.SimulateModel(model, model.Update(intents.FocusNextPane{}))
	require.True(t, bookmarkPaneVisible(model))
	require.False(t, bookmarkPaneFocused(model))

	test.SimulateModel(model, func() tea.Msg { return common.CloseViewMsg{} })

	assert.True(t, bookmarkPaneVisible(model))
	assert.True(t, model.revisions.IsFocused())
}

func Test_BookmarkPaneRevisionRowClick_FocusesRevisions(t *testing.T) {
	model := newBookmarkPaneModel(t, "main;.;true;false;false;false;abc123\n")

	test.SimulateModel(model, model.Update(intents.ToggleBookmarkPane{}))
	require.True(t, bookmarkPaneFocused(model))

	test.SimulateModel(model, func() tea.Msg { return revisions.ItemClickedMsg{Index: 0} })

	assert.False(t, bookmarkPaneFocused(model))
	assert.True(t, model.revisions.IsFocused())
}

func Test_BookmarkPaneConfirmation_UsesConfirmationScopeAndCancelKeepsPaneOpen(t *testing.T) {
	model := newBookmarkPaneModel(t, "main;.;true;false;false;false;abc123\n")

	test.SimulateModel(model, model.Update(intents.ToggleBookmarkPane{}))
	cmd, handled := model.HandleIntent(intents.BookmarkPaneDelete{})
	require.True(t, handled)
	test.SimulateModel(model, cmd)

	require.NotEmpty(t, model.dispatchScopes())
	assert.Equal(t, keybindings.ScopeName(actions.ScopeBookmarkPaneConfirmation), model.dispatchScopes()[0].Name)

	cmd, handled = model.HandleIntent(intents.Cancel{})
	require.True(t, handled)
	test.SimulateModel(model, cmd)

	assert.True(t, bookmarkPaneVisible(model))
	assert.True(t, bookmarkPaneFocused(model))
	require.NotEmpty(t, model.dispatchScopes())
	assert.Equal(t, keybindings.ScopeName(actions.ScopeBookmarkPane), model.dispatchScopes()[0].Name)
}

func Test_BookmarkPaneRowClick_FocusesBookmarkPane(t *testing.T) {
	model := newBookmarkPaneModel(t, "main;.;true;false;false;false;abc123\n")

	test.SimulateModel(model, model.Update(intents.ToggleBookmarkPane{}))
	test.SimulateModel(model, model.Update(intents.FocusNextPane{}))
	require.False(t, bookmarkPaneFocused(model))
	require.True(t, model.revisions.IsFocused())

	test.SimulateModel(model, func() tea.Msg { return bookmarkpane.ItemClickedMsg{Index: 0} })

	assert.True(t, bookmarkPaneFocused(model))
	assert.False(t, model.revisions.IsFocused())
}

func Test_BookmarkPaneMoveCancel_ReturnsFocusToBookmarkPane(t *testing.T) {
	model := newBookmarkPaneModel(t, "main;.;true;false;false;false;abc123\n")

	test.SimulateModel(model, model.Update(intents.ToggleBookmarkPane{}))
	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Text: "m", Code: 'm'}))
	require.False(t, bookmarkPaneFocused(model))
	require.False(t, model.revisions.InNormalMode())

	test.SimulateModel(model, model.Update(tea.KeyPressMsg{Code: tea.KeyEsc}))

	assert.True(t, model.revisions.InNormalMode())
	assert.True(t, bookmarkPaneFocused(model))
	assert.False(t, model.revisions.IsFocused())
}
