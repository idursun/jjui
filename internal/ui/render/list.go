package render

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
)

// viewPort defines the visible window of the list in list-line coordinates.
type viewPort struct {
	ViewRect  layout.Box // screen-space list box
	StartLine int        // list-line index at top of the viewport
}

// measureRequest is passed to items to ask how many lines they want to render.
type measureRequest struct {
	Index         int
	AvailableLine int // remaining viewport lines from the current item forward
}

// measureResult reports the desired line count for an item.
type measureResult struct {
	DesiredLine int // item would like to render this many lines
	MinLine     int // minimum number of lines the item can render
}

// span is the visible slice of an item mapped onto the screen.
type span struct {
	Index      int
	Rect       cellbuf.Rectangle
	LineOffset int
	LineCount  int
	ItemStart  int
	ItemEnd    int
}

type RenderItemFunc func(dl *DisplayContext, index int, rect cellbuf.Rectangle)

type measureItemFunc func(index int) int

type ClickMessage = tea.Msg

type ClickMessageFunc func(index int) ClickMessage

type ListRenderer struct {
	StartLine     int
	ScrollMsg     tea.Msg
	FirstRowIndex int
	LastRowIndex  int
}

func NewListRenderer(scrollMsg tea.Msg) *ListRenderer {
	return &ListRenderer{
		StartLine:     0,
		ScrollMsg:     scrollMsg,
		FirstRowIndex: -1,
		LastRowIndex:  -1,
	}
}

// Render renders visible items to the DisplayContext.
// Note: Scroll interaction registration is the caller's responsibility.
func (r *ListRenderer) Render(
	dl *DisplayContext,
	viewRect layout.Box,
	itemCount int,
	cursor int,
	ensureCursorVisible bool,
	measure measureItemFunc,
	render RenderItemFunc,
	clickMsg ClickMessageFunc,
) {
	if itemCount <= 0 {
		return
	}

	viewHeight := viewRect.R.Dy()
	totalLines := 0
	for i := 0; i < itemCount; i++ {
		totalLines += measure(i)
	}
	r.StartLine = ClampStartLine(r.StartLine, viewHeight, totalLines)

	viewport := viewPort{
		StartLine: r.StartLine,
		ViewRect:  viewRect,
	}

	if ensureCursorVisible && cursor >= 0 && cursor < itemCount {
		r.ensureCursorVisible(cursor, itemCount, viewRect.R.Dy(), measure)
		viewport.StartLine = r.StartLine
	}

	measureAdapter := func(req measureRequest) measureResult {
		if req.Index >= itemCount {
			return measureResult{DesiredLine: 0, MinLine: 0}
		}
		height := measure(req.Index)
		return measureResult{
			DesiredLine: height,
			MinLine:     height,
		}
	}

	spans, _ := layoutAll(viewport, itemCount, measureAdapter)
	if len(spans) > 0 {
		r.FirstRowIndex = spans[0].Index
		r.LastRowIndex = spans[len(spans)-1].Index
	} else {
		r.FirstRowIndex = -1
		r.LastRowIndex = -1
	}

	for _, span := range spans {
		if span.Index >= itemCount {
			continue
		}
		render(dl, span.Index, span.Rect)
		dl.AddInteraction(
			span.Rect,
			clickMsg(span.Index),
			InteractionClick,
			0,
		)
	}

}

func (r *ListRenderer) ensureCursorVisible(
	cursor int,
	itemCount int,
	viewportHeight int,
	measure measureItemFunc,
) {
	if cursor < 0 || cursor >= itemCount || viewportHeight <= 0 {
		return
	}

	cursorStart := 0
	for i := 0; i < cursor && i < itemCount; i++ {
		cursorStart += measure(i)
	}

	cursorHeight := measure(cursor)
	cursorEnd := cursorStart + cursorHeight

	start := r.StartLine
	if start < 0 {
		start = 0
	}

	viewportEnd := start + viewportHeight

	if cursorStart < start {
		r.StartLine = cursorStart
	} else if cursorEnd > viewportEnd {
		r.StartLine = cursorEnd - viewportHeight
		if r.StartLine < 0 {
			r.StartLine = 0
		}
	}
}

func (r *ListRenderer) SetScrollOffset(offset int) {
	r.StartLine = offset
}

func (r *ListRenderer) GetScrollOffset() int {
	return r.StartLine
}

func (r *ListRenderer) GetFirstRowIndex() int {
	return r.FirstRowIndex
}

func (r *ListRenderer) GetLastRowIndex() int {
	return r.LastRowIndex
}

// RegisterScroll registers a scroll interaction for the given view rect.
// Call this after Render if you want to enable mouse wheel scrolling.
func (r *ListRenderer) RegisterScroll(dl *DisplayContext, viewRect layout.Box) {
	if r.ScrollMsg == nil {
		return
	}
	dl.AddInteraction(
		viewRect.R,
		r.ScrollMsg,
		InteractionScroll,
		0,
	)
}

// layoutAll computes visible spans for list items and returns the total list height.
// Measure is called for all items in order so dynamic heights can be accounted for.
func layoutAll(
	viewport viewPort,
	itemCount int,
	measure func(measureRequest) measureResult,
) ([]span, int) {
	if itemCount <= 0 || viewport.ViewRect.R.Dy() <= 0 {
		return nil, 0
	}

	spans := make([]span, 0, 8)
	viewStart := viewport.StartLine
	viewEnd := viewport.StartLine + viewport.ViewRect.R.Dy()
	listY := 0

	for i := 0; i < itemCount; i++ {
		available := viewEnd - max(viewStart, listY)
		if available < 0 {
			available = 0
		}
		result := measure(measureRequest{
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
				spans = append(spans, span{
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

// ClampStartLine constrains a scroll start line to valid bounds.
// totalLines is the sum of all item heights; viewHeight is the visible area height.
func ClampStartLine(startLine, viewHeight, totalLines int) int {
	if startLine < 0 {
		startLine = 0
	}
	maxStart := totalLines - viewHeight
	if maxStart < 0 {
		maxStart = 0
	}
	if startLine > maxStart {
		startLine = maxStart
	}
	return startLine
}
