package render

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLayoutAllReturnsPartialSpan(t *testing.T) {
	spans, total := layoutAll(
		viewPort{
			ViewRect:  layout.NewBox(layout.Rect(0, 0, 10, 4)),
			StartLine: 3,
		},
		2,
		func(req measureRequest) measureResult {
			switch req.Index {
			case 0:
				return measureResult{DesiredLine: 5, MinLine: 5}
			case 1:
				return measureResult{DesiredLine: 2, MinLine: 2}
			default:
				return measureResult{}
			}
		},
	)

	require.Len(t, spans, 2)
	assert.Equal(t, 7, total)
	assert.Equal(t, 0, spans[0].Index)
	assert.Equal(t, 3, spans[0].LineOffset)
	assert.Equal(t, 2, spans[0].LineCount)
	assert.Equal(t, layout.Rect(0, 0, 10, 2), spans[0].Rect)
	assert.Equal(t, 1, spans[1].Index)
	assert.Equal(t, 0, spans[1].LineOffset)
	assert.Equal(t, 2, spans[1].LineCount)
	assert.Equal(t, layout.Rect(0, 2, 10, 2), spans[1].Rect)
}

func TestListRendererRenderClipsTopOfTallItem(t *testing.T) {
	renderer := NewListRenderer(nil)
	renderer.SetScrollOffset(3)

	dl := NewDisplayContext()
	renderer.Render(
		dl,
		layout.NewBox(layout.Rect(0, 0, 8, 2)),
		1,
		-1,
		false,
		func(_ int) int { return 5 },
		func(dl *DisplayContext, _ int, rect layout.Rectangle) {
			for i := range rect.Dy() {
				lineRect := layout.Rect(rect.Min.X, rect.Min.Y+i, rect.Dx(), 1)
				dl.AddDraw(lineRect, fmt.Sprintf("line %d", i+1), 0)
			}
		},
		func(index int, _ tea.Mouse) ClickMessage {
			return index
		},
	)

	out := dl.RenderToString(8, 2)
	assert.Contains(t, out, "line 4")
	assert.Contains(t, out, "line 5")
	assert.NotContains(t, out, "line 1")
	assert.NotContains(t, out, "line 2")
	assert.NotContains(t, out, "line 3")
}

func TestListRendererRenderClipsBottomOfTallItem(t *testing.T) {
	renderer := NewListRenderer(nil)

	dl := NewDisplayContext()
	renderer.Render(
		dl,
		layout.NewBox(layout.Rect(0, 0, 8, 3)),
		1,
		-1,
		false,
		func(_ int) int { return 5 },
		func(dl *DisplayContext, _ int, rect layout.Rectangle) {
			for i := range rect.Dy() {
				lineRect := layout.Rect(rect.Min.X, rect.Min.Y+i, rect.Dx(), 1)
				dl.AddDraw(lineRect, fmt.Sprintf("line %d", i+1), 0)
			}
		},
		func(index int, _ tea.Mouse) ClickMessage {
			return index
		},
	)

	out := dl.RenderToString(8, 3)
	assert.Contains(t, out, "line 1")
	assert.Contains(t, out, "line 2")
	assert.Contains(t, out, "line 3")
	assert.NotContains(t, out, "line 4")
	assert.NotContains(t, out, "line 5")
}

func TestListRendererRenderUsesVisibleHeightForBottomClippedItem(t *testing.T) {
	renderer := NewListRenderer(nil)

	var seenHeight int
	dl := NewDisplayContext()
	renderer.Render(
		dl,
		layout.NewBox(layout.Rect(0, 0, 8, 3)),
		1,
		-1,
		false,
		func(_ int) int { return 5 },
		func(_ *DisplayContext, _ int, rect layout.Rectangle) {
			seenHeight = rect.Dy()
		},
		func(index int, _ tea.Mouse) ClickMessage {
			return index
		},
	)

	assert.Equal(t, 3, seenHeight)
}

func TestListRendererKeepsItemInteractionsOfTopClippedItem(t *testing.T) {
	renderer := NewListRenderer(nil)
	renderer.SetScrollOffset(3)

	viewRect := layout.Rect(2, 1, 8, 2)
	dl := NewDisplayContext()
	renderer.Render(
		dl,
		layout.NewBox(viewRect),
		1,
		-1,
		false,
		func(_ int) int { return 5 },
		func(dl *DisplayContext, _ int, rect layout.Rectangle) {
			for i := range rect.Dy() {
				lineRect := layout.Rect(rect.Min.X, rect.Min.Y+i, rect.Dx(), 1)
				dl.AddInteraction(lineRect, fmt.Sprintf("line %d", i+1), InteractionClick, 0)
			}
		},
		func(index int, _ tea.Mouse) ClickMessage {
			return index
		},
	)

	click := func(x, y int) tea.Msg {
		msg, handled := dl.ProcessMouseEvent(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
		require.True(t, handled)
		return msg
	}
	assert.Equal(t, "line 4", click(viewRect.Min.X, viewRect.Min.Y))
	assert.Equal(t, "line 5", click(viewRect.Max.X-1, viewRect.Min.Y+1))

	_, handled := dl.ProcessMouseEvent(tea.MouseClickMsg{X: viewRect.Min.X, Y: viewRect.Min.Y - 1, Button: tea.MouseLeft})
	assert.False(t, handled, "scrolled-off lines must not be clickable outside the list")
}
