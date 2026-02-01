package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/git"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
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
	ctx := test.NewTestContext(commandRunner)

	model := NewUI(ctx)

	// Activate revset editing
	model.revsetModel.Update(revset.EditRevSetMsg{})
	assert.True(t, model.revsetModel.Editing, "revset should be in editing mode")

	// Trigger status update
	model.updateStatus()
	assert.Equal(t, "revset", model.status.Mode(), "status mode should be 'revset'")
	assert.Equal(t, model.revsetModel, model.status.Help(), "status help should be set to revset model")
}

// this test verifies that when `git` is activated and `status` is expanded,
// pressing `esc` closes expanded `status`
func Test_GitWithExpandedStatus_EscClosesStatusFirst(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := NewUI(ctx)

	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	// Directly set stacked to git model (simulates pressing 'g')
	gitModel := git.NewModel(ctx, jj.NewSelectedRevisions())
	test.SimulateModel(gitModel, gitModel.Init())
	model.stacked = gitModel
	assert.NotNil(t, model.stacked, "stacked (git) should be set")

	// Render to trigger status truncation detection
	_ = model.View()

	// Press '?' to expand status
	test.SimulateModel(model, test.Type("?"))
	assert.True(t, model.status.StatusExpanded(), "status should be expanded after pressing '?'")

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

	// Press 'esc' to close expanded status
	test.SimulateModel(model, test.Press(tea.KeyEscape))
	assert.False(t, model.status.StatusExpanded(), "status should be closed after pressing 'esc'")

	// Stacked (git) should still be open
	assert.NotNil(t, model.stacked, "stacked (git) should still be open after closing expanded status")
}
