package preview

import (
	"testing"

	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
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

func TestModel_YOffsetPersistence(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.Parent = common.NewViewNode(10, 50)
	model.SetFrame(cellbuf.Rect(0, 0, 10, 5)) // Small viewport to allow scrolling

	// Simulate selecting a file
	testFile := context.SelectedFile{
		ChangeId: "change1",
		CommitId: "commit1",
		File:     "file1.txt",
	}
	ctx.SelectedItem = testFile

	// Set content with many lines to allow scrolling
	longContent := ""
	for i := 1; i <= 20; i++ {
		longContent += "line" + string(rune('0'+i)) + "\n"
	}
	model.SetContent(longContent)

	model.Scroll(3)

	assert.Greater(t, model.view.YOffset, 0, "YOffset should be > 0 after scrolling")

	model.saveCurrentYOffset()
	assert.Equal(t, model.view.YOffset, model.fileYOffsets["file1.txt"], "YOffset should be saved for file1")

	testFile2 := context.SelectedFile{
		ChangeId: "change2",
		CommitId: "commit2",
		File:     "different-file.txt",
	}
	ctx.SelectedItem = testFile2

	model.reset()
	assert.Equal(t, 0, model.view.YOffset, "YOffset should be reset to 0")
	assert.Greater(t, model.fileYOffsets["file1.txt"], 0, "Original YOffset should be preserved")

	ctx.SelectedItem = testFile

	model.SetContent(longContent)
	expectedYOffset := model.fileYOffsets["file1.txt"]
	assert.Equal(t, expectedYOffset, model.view.YOffset, "YOffset should be restored to saved position for file1")
}

func TestModel_GlobalYOffsetPersistence(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)

	// Create first model and set scroll position
	model1 := New(ctx)
	model1.Parent = common.NewViewNode(10, 50)
	model1.SetFrame(cellbuf.Rect(0, 0, 10, 5))

	testFile := context.SelectedFile{
		ChangeId: "change1",
		CommitId: "commit1",
		File:     "global-test.txt",
	}
	ctx.SelectedItem = testFile

	longContent := ""
	for i := 1; i <= 20; i++ {
		longContent += "line" + string(rune('0'+i)) + "\n"
	}
	model1.SetContent(longContent)
	model1.Scroll(5)
	model1.saveCurrentYOffset()

	savedOffset := model1.fileYOffsets["global-test.txt"]
	assert.Greater(t, savedOffset, 0, "YOffset should be saved in first model")

	model2 := New(ctx)
	model2.Parent = common.NewViewNode(10, 50)
	model2.SetFrame(cellbuf.Rect(0, 0, 10, 5))

	assert.Equal(t, savedOffset, model2.fileYOffsets["global-test.txt"], "Second model should access global YOffset map")

	model2.SetContent(longContent)
	assert.Equal(t, savedOffset, model2.view.YOffset, "Second model should restore saved YOffset position")
}
