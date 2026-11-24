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

type boxRecorder struct {
	fakeContent
	box layout.Box
}

func (b *boxRecorder) ViewRect(_ *render.DisplayContext, box layout.Box) { b.box = box }

func TestSplitRendererFullSizeGivesSecondaryTheWholeBox(t *testing.T) {
	for _, vertical := range []bool{false, true} {
		state := NewSplitState(50)
		state.SetPercent(100)
		renderer := NewRenderer(state)
		primary, secondary := &boxRecorder{}, &boxRecorder{}
		renderer.Render(render.NewDisplayContext(), layout.NewBox(layout.Rect(0, 0, 20, 10)), primary, secondary, vertical)

		if vertical {
			assert.Equal(t, 0, primary.box.R.Dy())
			assert.Equal(t, 9, secondary.box.R.Dy())
		} else {
			assert.Equal(t, 0, primary.box.R.Dx())
			assert.Equal(t, 19, secondary.box.R.Dx())
		}
	}
}
