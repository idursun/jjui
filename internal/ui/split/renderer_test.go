package split

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
)

func TestSplitRendererSeparatorClickStartsDragAtMousePosition(t *testing.T) {
	renderer := NewRenderer(NewSplitState(50))
	dl := render.NewDisplayContext()
	renderer.Render(dl, layout.NewBox(layout.Rect(0, 0, 20, 10)), &fakeContent{}, &fakeContent{}, false)

	var drags []SplitDragMsg
	for x := range 20 {
		msg, _ := dl.ProcessMouseEvent(tea.MouseClickMsg{X: x, Y: 3, Button: tea.MouseLeft})
		if drag, ok := msg.(SplitDragMsg); ok {
			drags = append(drags, drag)
		}
	}

	if assert.Len(t, drags, 1, "only the separator column starts a drag") {
		assert.Same(t, renderer, drags[0].Renderer)
		assert.Equal(t, 3, drags[0].Y)
		assert.Positive(t, drags[0].X)
	}
}
