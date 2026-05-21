package revset

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel_Init(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	test.SimulateModel(model, model.Init())
}

func TestModel_Update_IntentDoesNotAlterCurrentRevsetDisplay(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.CurrentRevset = "current"
	ctx.DefaultRevset = "default"
	model := New(ctx)
	test.SimulateModel(model, model.Init())
	test.SimulateModel(model, func() tea.Msg { return intents.CompletionMove{Delta: -1} })
	assert.Contains(t, test.RenderImmediate(model, 80, 5), "current")
}

func TestModel_View_DisplaysCurrentRevset(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.CurrentRevset = "current"
	ctx.DefaultRevset = "default"
	model := New(ctx)
	assert.Contains(t, test.RenderImmediate(model, 80, 5), ctx.CurrentRevset)
}

func TestModel_View_KeepsCompletionListForPreviewedFunctionCompletion(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.editing = true
	model.userInput = "au"
	model.autoComplete.SetValue("au")
	model.updateCompletionItems()

	cmd := model.Update(intents.CompletionMove{Delta: 1})
	require.Nil(t, cmd)

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 100, 1)))
	buf := render.NewScreenBuffer(100, 5)
	dl.Render(buf)
	rendered := buf.Render()
	assert.Contains(t, rendered, "author(")
	assert.Contains(t, rendered, "function")
	assert.NotContains(t, rendered, "author(pattern)")
}

func TestModel_View_SelectedCompletionPaintsTextBackground(t *testing.T) {
	originalPalette := common.DefaultPalette
	palette := common.NewPalette()
	palette.Update(map[string]config.Color{
		"revset completion":                  {Bg: "black"},
		"revset completion text":             {Fg: "green"},
		"revset completion matched":          {Fg: "green", Bold: boolPtr(true)},
		"revset completion selected":         {Bg: "blue", Bold: boolPtr(true)},
		"revset completion selected text":    {Fg: "bright green"},
		"revset completion selected matched": {Bold: boolPtr(true)},
		"revset completion selected dimmed":  {Fg: "bright cyan"},
	})
	common.DefaultPalette = palette
	defer func() { common.DefaultPalette = originalPalette }()

	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.editing = true
	model.selectedIndex = 0
	model.completionItems = []CompletionItem{{
		Kind:        KindFunction,
		MatchedPart: "au",
		RestPart:    "thor",
	}}

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 1)))
	buf := render.NewScreenBuffer(40, 2)
	dl.Render(buf)

	cell := buf.CellAt(pillWidth+1, 1)
	_, wantBg := renderExpectedCellColors(t, lipgloss.NewStyle().Background(lipgloss.Color("4")).Render("a"))
	assert.Equal(t, wantBg, cell.Style.Bg)
}

func TestModel_Update_CommitsArgumentCompletionAndShowsRemoteValues(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin\nupstream\n"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.Update(intents.Edit{})
	model.autoComplete.SetValue("untracked_remote_bookmarks(re")
	model.userInput = "untracked_remote_bookmarks(re"
	model.updateCompletionItems()

	cmd := model.Update(intents.CompletionCycle{})
	require.Nil(t, cmd)

	assert.Equal(t, "untracked_remote_bookmarks(remote=", model.userInput)
	assert.Equal(t, "untracked_remote_bookmarks(remote=", model.autoComplete.Value())
	assert.NotEmpty(t, model.completionItems)
	assert.Equal(t, KindRemote, model.completionItems[0].Kind)
	assert.Equal(t, "origin", model.completionItems[0].InsertText)
}

func TestModel_Update_CyclesPreviewedBookmarkCompletionsUntilInputChanges(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.editing = true
	model.userInput = "fe"
	model.autoComplete.SetValue("fe")
	model.completionItems = []CompletionItem{
		{Name: "feature-a", InsertText: "feature-a", ReplaceStart: 0, Kind: KindBookmark, MatchedPart: "fe", RestPart: "ature-a"},
		{Name: "feature-b", InsertText: "feature-b", ReplaceStart: 0, Kind: KindBookmark, MatchedPart: "fe", RestPart: "ature-b"},
	}

	cmd := model.Update(intents.CompletionCycle{})
	require.Nil(t, cmd)
	assert.Equal(t, "feature-a", model.autoComplete.Value())
	assert.Equal(t, "fe", model.userInput)
	require.Len(t, model.completionItems, 2)

	cmd = model.Update(intents.CompletionCycle{})
	require.Nil(t, cmd)
	assert.Equal(t, "feature-b", model.autoComplete.Value())
	assert.Equal(t, "fe", model.userInput)
	require.Len(t, model.completionItems, 2)
}

func TestModel_View_ShowsRemoteCompletionsWhilePreviewingRemoteValue(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.editing = true
	model.userInput = "remote_bookmarks(remote="
	model.autoComplete.SetValue("remote_bookmarks(remote=origin")
	model.autoComplete.SignatureHelp = "remote_bookmarks([name_pattern[, [remote=]remote_pattern]]): All remote bookmark targets"
	model.selectedIndex = 0
	model.completionItems = []CompletionItem{
		{Name: "origin", InsertText: "origin", ReplaceStart: len(model.userInput), Kind: KindRemote},
		{Name: "upstream", InsertText: "upstream", ReplaceStart: len(model.userInput), Kind: KindRemote},
	}

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 100, 1)))
	buf := render.NewScreenBuffer(100, 5)
	dl.Render(buf)
	rendered := buf.Render()
	assert.Contains(t, rendered, "remote")
	assert.Contains(t, rendered, "origin")
	assert.NotContains(t, rendered, "All remote bookmark targets")
}

func TestModel_ApplyCompletion(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)

	tests := []struct {
		name           string
		input          string
		item           CompletionItem
		expectedOutput string
	}{
		{
			name:           "function without parameters",
			input:          "al",
			item:           CompletionItem{Name: "all", Kind: KindFunction, HasParameters: false},
			expectedOutput: "all()",
		},
		{
			name:           "function with parameters",
			input:          "au",
			item:           CompletionItem{Name: "author", Kind: KindFunction, HasParameters: true},
			expectedOutput: "author(",
		},
		{
			name:           "history item",
			input:          "a",
			item:           CompletionItem{Name: "ancestors()", Kind: KindHistory},
			expectedOutput: "ancestors()",
		},
		{
			name:           "alias without parameters",
			input:          "my",
			item:           CompletionItem{Name: "myalias", Kind: KindAlias, HasParameters: false},
			expectedOutput: "myalias",
		},
		{
			name:           "function with context before",
			input:          "present(@) | au",
			item:           CompletionItem{Name: "author", Kind: KindFunction, HasParameters: true},
			expectedOutput: "present(@) | author(",
		},
		{
			name:           "parameterless function with context",
			input:          "empty() & ",
			item:           CompletionItem{Name: "all", Kind: KindFunction, HasParameters: false},
			expectedOutput: "empty() & all()",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := model.applyCompletion(test.input, test.item)
			assert.Equal(t, test.expectedOutput, result, "applyCompletion(%q, %v) should return %q", test.input, test.item, test.expectedOutput)
		})
	}
}

func TestModel_Update_ApplyValidationErrorAndCancel(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.RevsetValidate("invalid")).SetError(errors.New("invalid revset"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.editing = true
	model.autoComplete.SetValue("invalid")

	cmd := model.Update(intents.Apply{})
	require.NotNil(t, cmd)
	msg := cmd()
	addMessage, ok := msg.(intents.AddMessage)
	require.True(t, ok, "apply should report invalid revset as flash message")
	assert.Equal(t, "invalid revset", addMessage.Text)
	assert.True(t, model.editing, "invalid apply should keep editing mode")

	cancelCmd := model.Update(intents.Cancel{})
	assert.Nil(t, cancelCmd)
	assert.False(t, model.editing, "cancel should exit editing mode")
}

func TestModel_Update_ApplyEmptyUsesDefaultRevset(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.RevsetValidate("assume-passed-from-cli"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.DefaultRevset = "assume-passed-from-cli"
	model := New(ctx)
	model.editing = true
	model.autoComplete.SetValue("")

	cmd := model.Update(intents.Apply{})
	require.NotNil(t, cmd)
	assert.False(t, model.editing, "successful apply should exit editing mode")

	var updated string
	test.SimulateModel(model, cmd, func(msg tea.Msg) {
		if update, ok := msg.(common.UpdateRevSetMsg); ok {
			updated = string(update)
		}
	})
	assert.Equal(t, ctx.DefaultRevset, updated, "empty apply should resolve to default revset")
}

func boolPtr(v bool) *bool { return &v }

func renderExpectedCellColors(t *testing.T, content string) (any, any) {
	t.Helper()
	dl := render.NewDisplayContext()
	dl.AddDraw(layout.Rect(0, 0, 1, 1), content, 0)
	buf := render.NewScreenBuffer(1, 1)
	dl.Render(buf)
	cell := buf.CellAt(0, 0)
	return cell.Style.Fg, cell.Style.Bg
}
