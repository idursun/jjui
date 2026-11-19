package preview

import (
	"testing"

	"github.com/charmbracelet/x/ansi"

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
			name:     "Scroll down is clamped when at bottom",
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
			.....
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
			model.wrap = false

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
	args := jj.TemplatedArgs(config.Current.Preview.RevisionCommand, map[string]string{
		jj.RevsetPlaceholder:       ctx.CurrentRevset,
		jj.ChangeIdPlaceholder:     selected.ChangeId,
		jj.CommitIdPlaceholder:     selected.CommitId,
		jj.PreviewWidthPlaceholder: "80",
	})
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

func TestModel_ViewWraps(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		content  string
		expected string
	}{
		{
			name:     "wraps at word boundaries",
			width:    10,
			height:   3,
			content:  "one two three four five",
			expected: "one two\nthree four\nfive",
		},
		{
			name:     "wraps long words",
			width:    5,
			height:   3,
			content:  "abcdefghijklm",
			expected: "abcde\nfghij\nklm",
		},
		{
			name:     "preserves explicit newlines and blank lines",
			width:    10,
			height:   4,
			content:  "one two three\n\nfour five",
			expected: "one two\nthree\n\nfour five",
		},
		{
			name:     "keeps hanging indentation",
			width:    14,
			height:   4,
			content:  "    alpha beta gamma delta\nshort",
			expected: "alpha beta\ngamma\ndelta\nshort",
		},
		{
			name:     "preserves wide characters across wraps",
			width:    5,
			height:   2,
			content:  "ab界界cd",
			expected: "ab界\n界cd",
		},
		{
			name:     "preserves grapheme clusters across wraps",
			width:    3,
			height:   2,
			content:  "abe\u0301cd",
			expected: "abe\u0301\ncd",
		},
		{
			name:     "expands tabs before wrapping",
			width:    6,
			height:   3,
			content:  "+\tfoo bar",
			expected: "+\nfoo\nbar",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := test.NewTestContext(test.NewTestCommandRunner(t))
			model := New(ctx)
			model.wrap = true
			model.SetContent(tc.content)

			rendered := test.Stripped(test.RenderImmediate(model, tc.width, tc.height))

			assert.Equal(t, tc.expected, rendered)
		})
	}
}

func TestWrapContent_HangingIndent(t *testing.T) {
	assert.Equal(t, "    alpha beta\n    gamma\n    delta", wrapContent("    alpha beta gamma delta", 14))
}

func TestWrapContent_DropsIndentThatLeavesTooLittleRoom(t *testing.T) {
	assert.Equal(t, "      ab\ncd ef gh", wrapContent("      ab cd ef gh", 10))
}

func TestWrapContent_LeavesShortLinesUntouched(t *testing.T) {
	content := "\x1b[31mred\x1b[0m\nplain"
	assert.Equal(t, content, wrapContent(content, 20))
}

func TestModel_ScrollWrappedContent(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.wrap = true
	model.SetContent("aaaa bbbb cccc dddd eeee ffff gggg hhhh")
	assert.Equal(t, "aaaa\nbbbb\ncccc\ndddd", test.Stripped(test.RenderImmediate(model, 4, 4)))

	model.Update(ScrollMsg{Delta: 1})
	assert.Equal(t, "bbbb\ncccc\ndddd\neeee", test.Stripped(test.RenderImmediate(model, 4, 4)))

	model.HalfPageDown()
	assert.Equal(t, "dddd\neeee\nffff\ngggg", test.Stripped(test.RenderImmediate(model, 4, 4)))

	model.PageDown()
	assert.Equal(t, "eeee\nffff\ngggg\nhhhh", test.Stripped(test.RenderImmediate(model, 4, 4)))

	model.HalfPageUp()
	assert.Equal(t, "cccc\ndddd\neeee\nffff", test.Stripped(test.RenderImmediate(model, 4, 4)))

	model.PageUp()
	assert.Equal(t, "aaaa\nbbbb\ncccc\ndddd", test.Stripped(test.RenderImmediate(model, 4, 4)))

	model.Update(ScrollMsg{Delta: 2, Horizontal: true})
	assert.Equal(t, "aaaa\nbbbb\ncccc\ndddd", test.Stripped(test.RenderImmediate(model, 4, 4)))
}

func TestModel_WrapIgnoresHorizontalScrollBeforeFirstRender(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.wrap = true

	model.SetContent("one two three four")
	model.ScrollHorizontal(3)

	assert.Equal(t, "one two\nthree four", test.Stripped(test.RenderImmediate(model, 10, 2)))
}

func TestModel_ResizeRewrapsOriginalContent(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.wrap = true
	model.SetContent("one two three four five six")

	assert.Equal(t, "one two\nthree four\nfive six", test.Stripped(test.RenderImmediate(model, 10, 8)))
	assert.Equal(t, "one two three four five six", test.Stripped(test.RenderImmediate(model, 30, 8)))
	assert.Equal(t, "one\ntwo\nthree\nfour\nfive\nsix", test.Stripped(test.RenderImmediate(model, 5, 8)))
}

func TestModel_ResizeClampsScroll(t *testing.T) {
	for _, size := range []struct{ width, height int }{{9, 2}, {4, 4}} {
		ctx := test.NewTestContext(test.NewTestCommandRunner(t))
		model := New(ctx)
		model.wrap = true
		model.SetContent("aaaa bbbb cccc dddd")
		test.RenderImmediate(model, 4, 2)
		model.Scroll(100)
		require.Equal(t, 2, model.YOffset())

		rendered := test.Stripped(test.RenderImmediate(model, size.width, size.height))

		assert.Zero(t, model.YOffset())
		assert.Contains(t, rendered, "aaaa")
		assert.Contains(t, rendered, "dddd")
	}
}

func TestModel_SetContentWrapsAtCurrentWidth(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.wrap = true
	test.RenderImmediate(model, 10, 3)

	model.SetContent("one two three four five")
	assert.Equal(t, "one two\nthree four\nfive", test.Stripped(test.RenderImmediate(model, 10, 3)))
}

func TestModel_WrappingPreservesANSIStyles(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.wrap = true
	const url = "https://example.com"
	model.SetContent(ansi.SetHyperlink(url) + "\x1b[31mone two three\x1b[0m" + ansi.ResetHyperlink() + "\nplain")

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 7, 2)))
	buf := render.NewScreenBuffer(7, 2)
	dl.Render(buf)

	assert.Equal(t, "one two\nthree", test.Stripped(ansi.Strip(buf.Render())))
	assert.Equal(t, ansi.Red, buf.CellAt(0, 0).Style.Fg)
	assert.Equal(t, ansi.Red, buf.CellAt(0, 1).Style.Fg)
	assert.Equal(t, url, buf.CellAt(0, 0).Link.URL)
	assert.Equal(t, url, buf.CellAt(0, 1).Link.URL)

	model.Scroll(1)
	dl = render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 7, 2)))
	buf = render.NewScreenBuffer(7, 2)
	dl.Render(buf)

	assert.Equal(t, "three\nplain", test.Stripped(ansi.Strip(buf.Render())))
	assert.Equal(t, ansi.Red, buf.CellAt(0, 0).Style.Fg)
	assert.Equal(t, url, buf.CellAt(0, 0).Link.URL)
	assert.Nil(t, buf.CellAt(0, 1).Style.Fg)
	assert.Empty(t, buf.CellAt(0, 1).Link.URL)
}

func TestModel_WrappingPreservesStyleOfIndentedLines(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.wrap = true
	model.SetContent("\x1b[33m    alpha beta gamma\x1b[0m")

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 12, 3)))
	buf := render.NewScreenBuffer(12, 3)
	dl.Render(buf)

	assert.Equal(t, "alpha\nbeta\ngamma", test.Stripped(ansi.Strip(buf.Render())))
	for y := range 3 {
		assert.Equal(t, ansi.Yellow, buf.CellAt(4, y).Style.Fg)
	}
}
