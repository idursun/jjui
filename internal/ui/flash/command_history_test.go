package flash

import (
	"fmt"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandHistory_NavigationAdjustsSelection(t *testing.T) {
	source := New()
	for range 6 {
		source.AddWithCommand("output", "jj cmd", nil)
	}

	history := source.NewHistory()
	assert.Equal(t, 5, history.selectedIndex)

	history.Update(intents.CommandHistoryNavigate{Delta: 1})
	history.Update(intents.CommandHistoryNavigate{Delta: 1})
	assert.Equal(t, 3, history.selectedIndex)

	history.Update(intents.CommandHistoryNavigate{Delta: -1})
	assert.Equal(t, 4, history.selectedIndex)
}

func TestCommandHistory_ViewOnlyShowsSelectedOutput(t *testing.T) {
	source := New()
	source.AddWithCommand("older-output", "jj older", nil)
	source.AddWithCommand("newer-output", "jj newer", nil)

	history := source.NewHistory()
	history.Update(intents.CommandHistoryNavigate{Delta: 1}) // select older

	dl := render.NewDisplayContext()
	box := layout.NewBox(layout.Rect(0, 0, 60, 12))
	history.ViewRect(dl, box)
	rendered := dl.RenderToString(box.R.Dx(), box.R.Dy())
	assert.Contains(t, rendered, "jj older")
	assert.Contains(t, rendered, "older-output")
	assert.Contains(t, rendered, "jj newer")
	assert.NotContains(t, rendered, "newer-output")
}

func TestCommandHistory_ViewKeepsSelectedEntryVisibleWhenItExpands(t *testing.T) {
	source := New()
	for i := range 6 {
		source.AddWithCommand(fmt.Sprintf("output-%d", i), fmt.Sprintf("jj cmd %d", i), nil)
	}
	source.AddWithCommand(strings.Repeat("expanded line\n", 4)+"expanded line", "jj expanded", nil)

	history := source.NewHistory()

	dl := render.NewDisplayContext()
	box := layout.NewBox(layout.Rect(0, 0, 60, 12))
	history.ViewRect(dl, box)
	rendered := dl.RenderToString(box.R.Dx(), box.R.Dy())

	assert.Contains(t, rendered, "jj expanded")
	assert.Contains(t, rendered, "expanded line")
}

func TestCommandHistory_ViewUsesAvailableHeightInsteadOfFixedWindow(t *testing.T) {
	source := New()
	for i := range 12 {
		source.AddWithCommand(fmt.Sprintf("output-%d", i), fmt.Sprintf("jj cmd %d", i), nil)
	}

	history := source.NewHistory()

	dl := render.NewDisplayContext()
	box := layout.NewBox(layout.Rect(0, 0, 60, 60))
	history.ViewRect(dl, box)
	rendered := dl.RenderToString(box.R.Dx(), box.R.Dy())

	assert.Contains(t, rendered, "jj cmd 0")
	assert.Contains(t, rendered, "jj cmd 11")
}

func TestCommandHistory_ViewDoesNotClipTopBorderOnExactFitBelowStatusBar(t *testing.T) {
	source := New()
	source.AddWithCommand("older-output", "jj older", nil)
	source.AddWithCommand("newer-output", "jj newer", nil)

	history := source.NewHistory()
	items := history.renderedItems(60-4, 100)
	require.Len(t, items, 2)

	totalHeight := 0
	for _, item := range items {
		totalHeight += item.h
	}

	dl := render.NewDisplayContext()
	box := layout.NewBox(layout.Rect(0, 0, 60, totalHeight+1))
	history.ViewRect(dl, box)
	rendered := dl.RenderToString(box.R.Dx(), box.R.Dy())

	firstLine := strings.Split(rendered, "\n")[0]
	assert.Contains(t, firstLine, "┌")
}

func TestCommandHistory_DeleteSelectedRemovesFromSourceAndLiveMessages(t *testing.T) {
	source := New()
	source.AddWithCommand("older-output", "jj older", nil)
	source.AddWithCommand("newer-output", "jj newer", nil)

	history := source.NewHistory()
	history.Update(intents.CommandHistoryNavigate{Delta: 1}) // select older
	history.Update(intents.CommandHistoryDeleteSelected{})

	if assert.Len(t, history.items, 1) {
		assert.Equal(t, "jj newer", history.items[0].Command)
	}

	snapshot := source.commandHistorySnapshot()
	if assert.Len(t, snapshot, 1) {
		assert.Equal(t, "jj newer", snapshot[0].Command)
	}
	assert.Equal(t, 1, source.LiveMessagesCount())
}

func TestCommandHistory_ViewFillsHistoryCardsBackground(t *testing.T) {
	originalPalette := common.DefaultPalette
	palette := common.NewPalette()
	palette.Update(map[string]config.Color{
		"flash text":    {Fg: "#ffffff", Bg: "#112244"},
		"flash success": {Fg: "#eafff2", Bg: "#1b5f46"},
		"flash matched": {Fg: "#ffee88", Bg: "#112244"},
	})
	common.DefaultPalette = palette
	defer func() { common.DefaultPalette = originalPalette }()

	source := New()
	source.AddWithCommand("older-output", "jj older", nil)
	source.AddWithCommand("newer-output", "jj newer", nil)

	history := source.NewHistory()
	dl := render.NewDisplayContext()
	box := layout.NewBox(layout.Rect(0, 0, 60, 12))
	history.ViewRect(dl, box)

	buf := uv.NewScreenBuffer(box.R.Dx(), box.R.Dy())
	dl.Render(buf)

	foundCardCell := false
	var borderBg any
	for y := 0; y < box.R.Dy(); y++ {
		for x := 0; x < box.R.Dx(); x++ {
			cell := buf.CellAt(x, y)
			if cell == nil {
				continue
			}
			if cell.Content == "┌" {
				borderBg = cell.Style.Bg
				assert.NotNil(t, cell.Style.Bg, "history card border should inherit a themed background")
				foundCardCell = true
			}
			if cell.Content == "j" {
				assert.NotNil(t, cell.Style.Bg, "history card cells should inherit a themed background")
				if borderBg != nil {
					assert.Equal(t, borderBg, cell.Style.Bg, "history command text should use the same surface background as the card")
				}
				foundCardCell = true
			}
		}
	}
	assert.True(t, foundCardCell, "expected to find at least one rendered history card cell")
}
