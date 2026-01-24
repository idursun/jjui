package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/revset"
	"github.com/stretchr/testify/assert"

	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/test"
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
