package render

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
)

// RenderItemFunc is called for each visible item. The implementor has full control
// over what gets drawn - they know their items, cursor position, and can call
// dl.AddDraw, dl.AddHighlight, etc. as needed.
type RenderItemFunc func(dl *DisplayList, index int, rect cellbuf.Rectangle)

// MeasureItemFunc returns the height (in lines) for the item at the given index.
type MeasureItemFunc func(index int) int

// ClickMessage is a type alias for tea.Msg used for click interactions.
type ClickMessage = tea.Msg

// ClickMessageFunc creates the message to send when item at index is clicked.
type ClickMessageFunc func(index int) ClickMessage

// ListRenderer provides generic list rendering with DisplayList.
// It handles layout calculation, viewport management, and mouse interaction
// registration. The actual item rendering is delegated to the caller.
type ListRenderer struct {
	StartLine int       // Current scroll position (line offset)
	ScrollMsg tea.Msg   // Message type for scroll events (must implement ScrollDeltaCarrier)
}

// NewListRenderer creates a new ListRenderer with the given scroll message type.
func NewListRenderer(scrollMsg tea.Msg) *ListRenderer {
	return &ListRenderer{
		StartLine: 0,
		ScrollMsg: scrollMsg,
	}
}

// Render renders visible items to the DisplayList.
// Uses viewRect.Min as the screen offset for interactions.
//
// Parameters:
//   - dl: The DisplayList to render to
//   - viewRect: The screen area for the list (absolute coordinates)
//   - itemCount: Total number of items in the list
//   - cursor: Current cursor position (used for ensureCursorVisible)
//   - ensureCursorVisible: Whether to adjust scroll to keep cursor in view
//   - measure: Function that returns height for item at index
//   - render: Function that renders item at index to the DisplayList
//   - clickMsg: Function that creates click message for item at index
func (r *ListRenderer) Render(
	dl *DisplayList,
	viewRect layout.Box,
	itemCount int,
	cursor int,
	ensureCursorVisible bool,
	measure MeasureItemFunc,
	render RenderItemFunc,
	clickMsg ClickMessageFunc,
) {
	r.RenderWithOffset(dl, viewRect, itemCount, cursor, ensureCursorVisible, measure, render, clickMsg, cellbuf.Pos(viewRect.R.Min.X, viewRect.R.Min.Y))
}

// RenderWithOffset renders visible items to the DisplayList with a custom screen offset.
// Use this when the list is embedded inside another component and the screen offset
// differs from the viewRect position.
//
// Parameters:
//   - dl: The DisplayList to render to
//   - viewRect: The area to render in (relative coordinates for the render buffer)
//   - itemCount: Total number of items in the list
//   - cursor: Current cursor position (used for ensureCursorVisible)
//   - ensureCursorVisible: Whether to adjust scroll to keep cursor in view
//   - measure: Function that returns height for item at index
//   - render: Function that renders item at index to the DisplayList
//   - clickMsg: Function that creates click message for item at index
//   - screenOffset: The offset to add for mouse interactions (absolute screen position)
func (r *ListRenderer) RenderWithOffset(
	dl *DisplayList,
	viewRect layout.Box,
	itemCount int,
	cursor int,
	ensureCursorVisible bool,
	measure MeasureItemFunc,
	render RenderItemFunc,
	clickMsg ClickMessageFunc,
	screenOffset cellbuf.Position,
) {
	if itemCount <= 0 {
		return
	}

	// Use the provided screen offset for interaction coordinates
	// Draws use the viewRect coordinates (relative to render buffer)
	// Interactions use screenOffset (absolute screen coordinates for mouse hit testing)
	screenOffsetX := screenOffset.X
	screenOffsetY := screenOffset.Y

	// Create viewport for layout calculations
	// Uses viewRect coordinates which may be relative to a parent component
	viewport := Viewport{
		StartLine: r.StartLine,
		ViewRect: layout.Box{
			R: cellbuf.Rect(viewRect.R.Min.X, viewRect.R.Min.Y, viewRect.R.Dx(), viewRect.R.Dy()),
		},
	}

	// Ensure cursor is visible by adjusting scroll position
	if ensureCursorVisible && cursor >= 0 && cursor < itemCount {
		r.ensureCursorVisible(cursor, itemCount, viewRect.R.Dy(), measure)
		viewport.StartLine = r.StartLine
	}

	// Calculate layout for visible items
	measureAdapter := func(req MeasureRequest) MeasureResult {
		if req.Index >= itemCount {
			return MeasureResult{DesiredLine: 0, MinLine: 0}
		}
		height := measure(req.Index)
		return MeasureResult{
			DesiredLine: height,
			MinLine:     height,
		}
	}

	spans, _ := LayoutAll(viewport, itemCount, measureAdapter)

	// Render each visible item (using relative coordinates for draws)
	for _, span := range spans {
		if span.Index >= itemCount {
			continue
		}
		render(dl, span.Index, span.Rect)
	}

	// Add click interactions for each visible item
	// span.Rect is already in absolute screen coordinates (includes viewRect.R.Min offset from LayoutAll)
	for _, span := range spans {
		if span.Index >= itemCount {
			continue
		}

		dl.AddInteraction(
			span.Rect,
			clickMsg(span.Index),
			InteractionClick,
			0,
		)
	}

	// Add scrollable region for the entire viewport (using absolute screen coordinates)
	if r.ScrollMsg != nil {
		scrollRect := cellbuf.Rect(
			screenOffsetX,
			screenOffsetY,
			viewRect.R.Dx(),
			viewRect.R.Dy(),
		)
		dl.AddInteraction(
			scrollRect,
			r.ScrollMsg,
			InteractionScroll,
			0,
		)
	}
}

// ensureCursorVisible adjusts StartLine to keep the cursor visible in the viewport.
func (r *ListRenderer) ensureCursorVisible(
	cursor int,
	itemCount int,
	viewportHeight int,
	measure MeasureItemFunc,
) {
	if cursor < 0 || cursor >= itemCount || viewportHeight <= 0 {
		return
	}

	// Calculate the line position where the cursor item starts
	cursorStart := 0
	for i := 0; i < cursor && i < itemCount; i++ {
		cursorStart += measure(i)
	}

	// Calculate the height of the cursor item
	cursorHeight := measure(cursor)
	cursorEnd := cursorStart + cursorHeight

	start := r.StartLine
	if start < 0 {
		start = 0
	}

	viewportEnd := start + viewportHeight

	// Only adjust if cursor is outside the current viewport
	if cursorStart < start {
		// Cursor is above viewport, scroll up
		r.StartLine = cursorStart
	} else if cursorEnd > viewportEnd {
		// Cursor is below viewport, scroll down
		r.StartLine = cursorEnd - viewportHeight
		if r.StartLine < 0 {
			r.StartLine = 0
		}
	}
}

// SetScrollOffset sets the scroll position.
func (r *ListRenderer) SetScrollOffset(offset int) {
	r.StartLine = offset
}

// GetScrollOffset returns the current scroll position.
func (r *ListRenderer) GetScrollOffset() int {
	return r.StartLine
}
