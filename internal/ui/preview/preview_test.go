package preview

import (
	"testing"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"

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

			model.previewAtBottom = tc.atBottom
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

	selected := common.SelectedRevision{ChangeId: "change", CommitId: "commit"}
	args := jj.TemplatedArgs(config.Current.Preview.RevisionCommand, map[string]string{
		jj.RevsetPlaceholder:       ctx.CurrentRevset,
		jj.ChangeIdPlaceholder:     selected.ChangeId,
		jj.CommitIdPlaceholder:     selected.CommitId,
		jj.PreviewWidthPlaceholder: "0",
	})
	commandRunner.Expect(args).SetOutput([]byte("auto preview"))

	model.Update(intents.PreviewShow{Content: "manual"})
	cmd := model.Update(common.SelectionChangedMsg{Item: selected})
	require.NotNil(t, cmd)

	test.SimulateModel(model, cmd)

	assert.Equal(t, "auto preview", model.content)
}
