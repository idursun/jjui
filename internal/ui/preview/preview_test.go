package preview

import (
	"testing"

	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestModel_Init(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.Parent = common.NewViewNode(10, 10)

	test.SimulateModel(model, model.Init())
}

func TestModel_View(t *testing.T) {
	tests := []struct {
		name     string
		scrollBy cellbuf.Position
		atBottom bool
		width    int
		height   int
		content  string
		expected string
	}{
		{
			name:     "clips",
			scrollBy: cellbuf.Position{},
			width:    5,
			height:   2,
			content: test.Stripped(`
			+++++..
			+abcde.
			+++++..
			`),
			expected: test.Stripped(`
			│++++
			│+abc
			`),
		},
		{
			name:     "clips when at bottom",
			scrollBy: cellbuf.Position{},
			atBottom: true,
			width:    5,
			height:   3,
			content: test.Stripped(`
			+++++..
			+abcde.
			+++++..
			`),
			expected: test.Stripped(`
			─────
			+++++
			+abcd
			`),
		},
		{
			name:     "Scroll by down and right",
			scrollBy: cellbuf.Position{X: 1, Y: 1},
			width:    5,
			height:   2,
			content: test.Stripped(`
			.......
			.abcde.
			.......
			`),
			expected: test.Stripped(`
			│abcd
			│....
			`),
		},
		{
			name:     "Scroll down when at bottom",
			scrollBy: cellbuf.Position{X: 0, Y: 1},
			atBottom: true,
			width:    5,
			height:   3,
			content: test.Stripped(`
			.......
			.abcde.
			.......
			`),
			expected: test.Stripped(`
			─────
			.abcd
			.....
			`),
		},
		{
			name:     "Scroll 2 right when at bottom",
			scrollBy: cellbuf.Position{X: 2, Y: 0},
			atBottom: true,
			width:    5,
			height:   3,
			content: test.Stripped(`
			.......
			.abcde.
			.......
			`),
			expected: test.Stripped(`
			─────
			.....
			bcde.
			`),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := test.NewTestContext(test.NewTestCommandRunner(t))

			model := New(ctx)
			model.Parent = common.NewViewNode(10, 10)

			model.previewAtBottom = tc.atBottom
			model.SetFrame(cellbuf.Rect(0, 0, tc.width, tc.height))
			model.SetContent(tc.content)
			if tc.scrollBy.X > 0 {
				model.ScrollHorizontal(tc.scrollBy.X)
			}
			if tc.scrollBy.Y > 0 {
				model.Scroll(tc.scrollBy.Y)
			}
			v := test.Stripped(model.View())

			assert.Equal(t, tc.expected, v)
		})
	}
}
