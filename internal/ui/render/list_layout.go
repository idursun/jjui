package render

import (
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
)

// Viewport defines the visible window of the list in list-line coordinates.
type Viewport struct {
	ViewRect  layout.Box // screen-space list box
	StartLine int        // list-line index at top of the viewport
}

// MeasureRequest is passed to items to ask how many lines they want to render.
type MeasureRequest struct {
	Index         int
	AvailableLine int // remaining viewport lines from the current item forward
}

// MeasureResult reports the desired line count for an item.
type MeasureResult struct {
	DesiredLine int // item would like to render this many lines
	MinLine     int // minimum number of lines the item can render
}

// Span is the visible slice of an item mapped onto the screen.
type Span struct {
	Index      int
	Rect       cellbuf.Rectangle
	LineOffset int
	LineCount  int
	ItemStart  int
	ItemEnd    int
}

// LayoutAll computes visible spans for list items and returns the total list height.
// Measure is called for all items in order so dynamic heights can be accounted for.
func LayoutAll(
	viewport Viewport,
	itemCount int,
	measure func(MeasureRequest) MeasureResult,
) ([]Span, int) {
	if itemCount <= 0 || viewport.ViewRect.R.Dy() <= 0 {
		return nil, 0
	}

	spans := make([]Span, 0, 8)
	viewStart := viewport.StartLine
	viewEnd := viewport.StartLine + viewport.ViewRect.R.Dy()
	listY := 0

	for i := 0; i < itemCount; i++ {
		available := viewEnd - max(viewStart, listY)
		if available < 0 {
			available = 0
		}
		result := measure(MeasureRequest{
			Index:         i,
			AvailableLine: available,
		})

		height := result.DesiredLine
		if height < result.MinLine {
			height = result.MinLine
		}
		if height < 0 {
			height = 0
		}

		itemStart := listY
		itemEnd := listY + height

		if itemEnd > viewStart && itemStart < viewEnd {
			overlapStart := max(itemStart, viewStart)
			overlapEnd := min(itemEnd, viewEnd)
			visible := overlapEnd - overlapStart
			if visible > 0 {
				y := viewport.ViewRect.R.Min.Y + (overlapStart - viewStart)
				rect := cellbuf.Rect(
					viewport.ViewRect.R.Min.X,
					y,
					viewport.ViewRect.R.Dx(),
					visible,
				)
				spans = append(spans, Span{
					Index:      i,
					Rect:       rect,
					LineOffset: overlapStart - itemStart,
					LineCount:  visible,
					ItemStart:  itemStart,
					ItemEnd:    itemEnd,
				})
			}
		}

		listY = itemEnd
	}

	return spans, listY
}
