package preview

import (
	"errors"
	"testing"

	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"

	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type selectionStateProvider struct{ snapshot common.SelectionSnapshot }

func (p selectionStateProvider) Selection() common.SelectionSnapshot { return p.snapshot }
func (p selectionStateProvider) QueryState(string) (any, bool)       { return nil, false }

func TestModel_Init(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)

	test.SimulateModel(model, model.Init())
}

func TestModel_View(t *testing.T) {
	tests := []struct {
		name     string
		scrollBy layout.Position
		atBottom bool
		width    int
		height   int
		content  string
		expected string
	}{
		{
			name:     "clips",
			scrollBy: layout.Position{},
			width:    5,
			height:   2,
			content: test.Stripped(`
			+++++..
			+abcde.
			+++++..
			`),
			expected: test.Stripped(`
			+++++
			+abcd
			`),
		},
		{
			name:     "clips when at bottom",
			scrollBy: layout.Position{},
			atBottom: true,
			width:    5,
			height:   3,
			content: test.Stripped(`
			+++++..
			+abcde.
			+++++..
			`),
			expected: test.Stripped(`
			+++++
			+abcd
			+++++
			`),
		},
		{
			name:     "Scroll by down and right",
			scrollBy: layout.Position{X: 1, Y: 1},
			width:    5,
			height:   2,
			content: test.Stripped(`
			.......
			.abcde.
			.......
			`),
			expected: test.Stripped(`
			abcde
			.....
			`),
		},
		{
			name:     "Scroll down when at bottom",
			scrollBy: layout.Position{X: 0, Y: 1},
			atBottom: true,
			width:    5,
			height:   3,
			content: test.Stripped(`
			.......
			.abcde.
			.......
			`),
			expected: test.Stripped(`
			.abcd
			.....
			`),
		},
		{
			name:     "Scroll 2 right when at bottom",
			scrollBy: layout.Position{X: 2, Y: 0},
			atBottom: true,
			width:    5,
			height:   3,
			content: test.Stripped(`
			.......
			.abcde.
			.......
			`),
			expected: test.Stripped(`
			.....
			bcde.
			.....
			`),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := test.NewTestContext(test.NewTestCommandRunner(t))

			model := New(ctx)

			model.SetContent(tc.content)
			if tc.scrollBy.X > 0 {
				model.ScrollHorizontal(tc.scrollBy.X)
			}
			if tc.scrollBy.Y > 0 {
				model.Scroll(tc.scrollBy.Y)
			}
			v := test.Stripped(test.RenderImmediate(model, tc.width, tc.height))

			assert.Equal(t, tc.expected, v)
		})
	}
}

func TestUpdate_PreviewShowResetsScroll(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)

	model.SetContent("first line\nsecond line\nthird line")
	model.Scroll(2)
	model.ScrollHorizontal(2)

	cmd := model.Update(intents.PreviewShow{Content: "updated"})
	require.Nil(t, cmd)

	assert.Equal(t, 0, model.view.YOffset())
	assert.Equal(t, 0, model.view.XOffset())
	assert.Equal(t, "updated", model.content)
}

func TestUpdate_PreviewShowDoesNotBreakSelectionRefresh(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.CurrentRevset = "all()"
	model := New(ctx)
	model.ViewRect(render.NewDisplayContext(), layout.NewBox(layout.Rect(0, 0, 80, 3)))

	selected := common.SelectedRevision{ChangeId: "change", CommitId: "commit"}
	ctx.SetStateProvider(selectionStateProvider{snapshot: common.SelectionSnapshot{Highlighted: selected}})
	args := []string{"show", "--color", "always", "-r", "change"}
	commandRunner.Expect(args).SetOutput([]byte("line 1\nline 2\nline 3\nline 4\nline 5"))

	model.Update(intents.PreviewShow{Content: "manual"})
	cmd := model.Update(common.RefreshMsg{})
	require.NotNil(t, cmd)

	test.SimulateModel(model, cmd)

	assert.Equal(t, "line 1\nline 2\nline 3\nline 4\nline 5", model.content)

	model.Scroll(2)
	assert.Equal(t, 2, model.YOffset())

	cmd = model.Update(common.RefreshMsg{})
	require.NotNil(t, cmd)

	test.SimulateModel(model, cmd)

	assert.Equal(t, "line 1\nline 2\nline 3\nline 4\nline 5", model.content)
	assert.Equal(t, 2, model.YOffset())
}

func TestSetContent_ExpandsTabsUsingTabStops(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)

	model.SetContent("+\tfoo")

	rendered := test.Stripped(test.RenderImmediate(model, 12, 1))
	assert.Equal(t, "+   foo", rendered)
}

func TestSetContent_ResetsTabStopsAfterNewlines(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)

	model.SetContent("a\tb\nab\tc")

	rendered := test.Stripped(test.RenderImmediate(model, 12, 2))
	assert.Equal(t, "a   b\nab  c", rendered)
}

func refreshWith(t *testing.T, commandRunner *test.CommandRunner, item common.SelectedItem) *Model {
	t.Helper()
	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.ViewRect(render.NewDisplayContext(), layout.NewBox(layout.Rect(0, 0, 80, 5)))
	ctx.SetStateProvider(selectionStateProvider{snapshot: common.SelectionSnapshot{Highlighted: item}})
	cmd := model.Update(common.RefreshMsg{})
	require.NotNil(t, cmd)
	test.SimulateModel(model, cmd)
	return model
}

func TestRefresh_BookmarkObjectRunsBookmarkCommand(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()
	commandRunner.Expect([]string{"diff", "--color", "always", "--from", "fork_point(trunk() | main@origin)", "--to", "main@origin"}).
		SetOutput([]byte("bookmark diff"))

	model := refreshWith(t, commandRunner, common.SelectedRevision{
		ChangeId: "change", CommitId: "commit", ObjectKind: "bookmark", ObjectValue: "main@origin",
	})

	assert.Equal(t, "bookmark diff", model.content)
	assert.Equal(t, "jj diff --color always --from fork_point(trunk() | main@origin) --to main@origin", model.commandLine)
}

func TestRefresh_KindWithoutConfigFallsBackToNode(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()
	commandRunner.Expect([]string{"show", "--color", "always", "-r", "change"}).SetOutput([]byte("node show"))

	model := refreshWith(t, commandRunner, common.SelectedRevision{
		ChangeId: "change", CommitId: "commit", ObjectKind: "author", ObjectValue: "someone",
	})

	assert.Equal(t, "node show", model.content)
}

func TestRefresh_CommandErrorIsShown(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()
	commandRunner.Expect([]string{"show", "--color", "always", "-r", "change"}).SetError(errors.New("Error: boom\n"))

	model := refreshWith(t, commandRunner, common.SelectedRevision{ChangeId: "change", CommitId: "commit"})

	assert.Equal(t, "Error: boom", model.content)
}

func TestView_CommandLineIsPinnedAboveContent(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.SetContent("one\ntwo\nthree")
	model.commandLine = "jj show -r abcdef"
	model.Scroll(1)

	rendered := test.Stripped(test.RenderImmediate(model, 10, 3))
	assert.Equal(t, "jj show -r\ntwo\nthree", rendered)

	test.RenderImmediate(model, 10, 1)
	test.RenderImmediate(model, 10, 0)
}

func TestUpdate_PreviewShowClearsCommandLine(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.commandLine = "jj show"

	model.Update(intents.PreviewShow{Content: "manual"})

	assert.Empty(t, model.commandLine)
}
