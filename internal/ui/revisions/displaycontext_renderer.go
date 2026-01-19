package revisions

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

// DisplayContextRenderer renders the revisions list using the DisplayContext approach
type DisplayContextRenderer struct {
	listRenderer  *render.ListRenderer
	selections    map[string]bool
	textStyle     lipgloss.Style
	dimmedStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	matchedStyle  lipgloss.Style
}

// NewDisplayContextRenderer creates a new DisplayContext-based renderer
func NewDisplayContextRenderer(textStyle, dimmedStyle, selectedStyle, matchedStyle lipgloss.Style) *DisplayContextRenderer {
	return &DisplayContextRenderer{
		listRenderer:  render.NewListRenderer(ViewportScrollMsg{}),
		textStyle:     textStyle,
		dimmedStyle:   dimmedStyle,
		selectedStyle: selectedStyle,
		matchedStyle:  matchedStyle,
	}
}

// SetSelections sets the selected revisions for rendering checkboxes
func (r *DisplayContextRenderer) SetSelections(selections map[string]bool) {
	r.selections = selections
}

// Render renders the revisions list to a DisplayContext
func (r *DisplayContextRenderer) Render(
	dl *render.DisplayContext,
	items []parser.Row,
	cursor int,
	viewRect layout.Box,
	operation operations.Operation,
	ensureCursorVisible bool,
) {
	if len(items) == 0 {
		return
	}

	// Measure function - calculates height for each item
	measure := func(index int) int {
		item := items[index]
		isSelected := index == cursor
		return r.calculateItemHeight(item, isSelected, operation)
	}

	// Screen offset for interactions (absolute screen position)
	screenOffset := cellbuf.Pos(viewRect.R.Min.X, viewRect.R.Min.Y)

	// Render function - renders each visible item
	renderItem := func(dl *render.DisplayContext, index int, rect cellbuf.Rectangle) {
		item := items[index]
		isSelected := index == cursor

		// Render the item content
		r.renderItemToDisplayContext(dl, item, rect, isSelected, operation, screenOffset)

		// Add highlights for selected item (only for Highlightable lines)
		if isSelected {
			r.addHighlights(dl, item, rect, operation)
		}
	}

	// Click message factory
	clickMsg := func(index int) render.ClickMessage {
		return ItemClickedMsg{Index: index}
	}

	// Use the generic list renderer
	r.listRenderer.Render(
		dl,
		viewRect,
		len(items),
		cursor,
		ensureCursorVisible,
		measure,
		renderItem,
		clickMsg,
	)

	// Register scroll only when no overlay operation is active
	if overlay, ok := operation.(common.Overlay); !ok || !overlay.IsOverlay() {
		r.listRenderer.RegisterScroll(dl, viewRect)
	}
}

// addHighlights adds highlight effects for lines with Highlightable flag
func (r *DisplayContextRenderer) addHighlights(
	dl *render.DisplayContext,
	item parser.Row,
	rect cellbuf.Rectangle,
	operation operations.Operation,
) {
	y := rect.Min.Y

	// Account for operation "before" lines
	if operation != nil {
		before := operation.Render(item.Commit, operations.RenderPositionBefore)
		if before != "" {
			y += strings.Count(before, "\n") + 1
		}
	}

	// Add highlights only for lines with Highlightable flag
	for _, line := range item.Lines {
		if line.Flags&parser.Highlightable == parser.Highlightable {
			lineRect := cellbuf.Rect(rect.Min.X, y, rect.Dx(), 1)
			dl.AddHighlight(lineRect, r.selectedStyle, 1)
		}
		y++
	}
}

// calculateItemHeight calculates the height of an item in lines
func (r *DisplayContextRenderer) calculateItemHeight(
	item parser.Row,
	isSelected bool,
	operation operations.Operation,
) int {
	// Base height from the item's lines
	height := len(item.Lines)

	// Add operation height if item is selected and operation exists
	if isSelected && operation != nil {
		// Count lines in before section
		// Use DesiredHeight if available for DisplayContext operations
		desired := operation.DesiredHeight(item.Commit, operations.RenderPositionBefore)
		if desired > 0 {
			height += desired
		} else {
			before := operation.Render(item.Commit, operations.RenderPositionBefore)
			if before != "" {
				height += strings.Count(before, "\n") + 1
			}
		}

		// Count lines in overlay section (replaces description)
		overlay := operation.Render(item.Commit, operations.RenderOverDescription)
		if overlay != "" {
			// When overlay exists, we need to calculate more carefully
			overlayLines := strings.Count(overlay, "\n") + 1

			// Count how many description lines would be replaced
			descLines := 0
			for _, line := range item.Lines {
				if line.Flags&parser.Highlightable == parser.Highlightable &&
					line.Flags&parser.Revision != parser.Revision {
					descLines++
				}
			}

			// Adjust height: remove replaced description lines, add overlay lines
			height = height - descLines + overlayLines
		}

		// Count lines in after section
		// Use DesiredHeight if available for DisplayContext operations
		desiredAfter := operation.DesiredHeight(item.Commit, operations.RenderPositionAfter)
		if desiredAfter > 0 {
			height += desiredAfter
		} else {
			after := operation.Render(item.Commit, operations.RenderPositionAfter)
			if after != "" {
				height += strings.Count(after, "\n") + 1
			}
		}
	}

	return height
}

// renderItemToDisplayContext renders a single item to the DisplayContext
func (r *DisplayContextRenderer) renderItemToDisplayContext(
	dl *render.DisplayContext,
	item parser.Row,
	rect cellbuf.Rectangle,
	isSelected bool,
	operation operations.Operation,
	screenOffset cellbuf.Position,
) {
	y := rect.Min.Y

	// Create an item renderer for this item
	ir := itemRenderer{
		row:           item,
		isHighlighted: isSelected,
		op:            operation,
	}

	// Check if this revision is selected (for checkbox)
	if item.Commit != nil && r.selections != nil {
		ir.isChecked = r.selections[item.Commit.ChangeId]
	}

	// Setup styles from renderer
	ir.selectedStyle = r.selectedStyle
	ir.textStyle = r.textStyle
	ir.dimmedStyle = r.dimmedStyle
	ir.matchedStyle = r.matchedStyle

	// Handle operation rendering for before section
	if isSelected && operation != nil {
		before := operation.Render(item.Commit, operations.RenderPositionBefore)
		if before != "" {
			// Render before section
			lines := strings.Split(before, "\n")
			extended := parser.GraphGutter{}
			if item.Previous != nil {
				extended = item.Previous.Extend()
			}

			for _, line := range lines {
				if y >= rect.Max.Y {
					break
				}

				lineRect := cellbuf.Rect(rect.Min.X, y, rect.Dx(), 1)
				r.renderOperationLine(dl, lineRect, extended, line)
				y++
			}
		}
	}

	// Handle main content and description overlay
	descriptionOverlay := ""
	if isSelected && operation != nil {
		descriptionOverlay = operation.Render(item.Commit, operations.RenderOverDescription)
	}

	// If we render an "after" operation (e.g. details) we defer elided markers so
	// the operation can be inserted before elided markers (keeping them "between" commits).
	insertAfterBeforeElided := false
	if isSelected && operation != nil {
		if desiredAfter := operation.DesiredHeight(item.Commit, operations.RenderPositionAfter); desiredAfter > 0 {
			insertAfterBeforeElided = true
		} else if after := operation.Render(item.Commit, operations.RenderPositionAfter); after != "" {
			insertAfterBeforeElided = true
		}
	}

	renderAfter := func() {
		// Handle operation rendering for after section
		if !isSelected || operation == nil || item.Commit.IsRoot() {
			return
		}

		// Calculate extended gutter and its width for proper indentation
		extended := item.Extend()
		gutterWidth := 0
		for _, segment := range extended.Segments {
			gutterWidth += lipgloss.Width(segment.Text)
		}

		// Create content rect offset by gutter width
		contentRect := cellbuf.Rect(rect.Min.X+gutterWidth, y, rect.Dx()-gutterWidth, rect.Max.Y-y)

		// Screen offset for interactions - contentRect already includes the gutter offset
		// and y position, so just pass the parent's screenOffset through
		contentScreenOffset := screenOffset

		// Render the operation content
		height := operation.RenderToDisplayContext(dl, item.Commit, operations.RenderPositionAfter, contentRect, contentScreenOffset)

		if height > 0 {
			// Render gutters for each line
			for i := 0; i < height; i++ {
				gutterContent := r.renderGutter(extended)
				gutterRect := cellbuf.Rect(rect.Min.X, y+i, gutterWidth, 1)
				dl.AddDraw(gutterRect, gutterContent, 0)
			}
			y += height
			return
		}

		// Fall back to string-based rendering
		after := operation.Render(item.Commit, operations.RenderPositionAfter)
		if after == "" {
			return
		}

		lines := strings.Split(after, "\n")
		for _, line := range lines {
			if y >= rect.Max.Y {
				break
			}

			lineRect := cellbuf.Rect(rect.Min.X, y, rect.Dx(), 1)
			r.renderOperationLine(dl, lineRect, extended, line)
			y++
		}
	}

	// Render main lines
	descriptionRendered := false
	afterRendered := false

	for i := 0; i < len(item.Lines); i++ {
		line := item.Lines[i]

		// If an "after" operation is active, render it before the first elided marker
		// line so that elided markers stay visually between revisions.
		if insertAfterBeforeElided && !afterRendered && line.Flags&parser.Elided == parser.Elided {
			renderAfter()
			afterRendered = true
		}

		// Skip elided lines when we have description overlay
		if line.Flags&parser.Elided == parser.Elided && descriptionOverlay != "" {
			continue
		}

		// Handle description overlay
		if descriptionOverlay != "" && !descriptionRendered &&
			line.Flags&parser.Highlightable == parser.Highlightable &&
			line.Flags&parser.Revision != parser.Revision {

			// Render description overlay
			overlayLines := strings.Split(descriptionOverlay, "\n")
			for _, overlayLine := range overlayLines {
				if y >= rect.Max.Y {
					break
				}

				lineRect := cellbuf.Rect(rect.Min.X, y, rect.Dx(), 1)
				r.renderOperationLine(dl, lineRect, line.Gutter, overlayLine)
				y++
			}

			descriptionRendered = true
			// Skip remaining description lines
			for i < len(item.Lines) && item.Lines[i].Flags&parser.Highlightable == parser.Highlightable {
				i++
			}
			i-- // Adjust because loop will increment
			continue
		}

		// Render normal line
		if y >= rect.Max.Y {
			break
		}

		lineRect := cellbuf.Rect(rect.Min.X, y, rect.Dx(), 1)
		dl.AddFill(lineRect, ' ', lipgloss.NewStyle(), 0)
		tb := dl.Text(lineRect.Min.X, lineRect.Min.Y, 0)
		ir.renderLine(tb, line)
		tb.Done()
		y++
	}

	// Render operation after section if it wasn't already inserted before elided markers.
	if !afterRendered {
		renderAfter()
	}
}

// renderLine writes a line into a TextBuilder (helper for itemRenderer)
func (ir *itemRenderer) renderLine(tb *render.TextBuilder, line *parser.GraphRowLine) {
	// Render gutter (no tracer support for now)
	for _, segment := range line.Gutter.Segments {
		style := segment.Style.Inherit(ir.textStyle)
		tb.Styled(segment.Text, style)
	}

	// Add checkbox and operation content before ChangeID
	if line.Flags&parser.Revision == parser.Revision {
		if ir.isChecked {
			tb.Styled("✓ ", ir.selectedStyle)
		}
		beforeChangeID := ir.op.Render(ir.row.Commit, operations.RenderBeforeChangeId)
		if beforeChangeID != "" {
			tb.Write(beforeChangeID)
		}
	}

	// Render segments
	beforeCommitID := ""
	if ir.op != nil {
		beforeCommitID = ir.op.Render(ir.row.Commit, operations.RenderBeforeCommitId)
	}

	for _, segment := range line.Segments {
		if beforeCommitID != "" && segment.Text == ir.row.Commit.CommitId {
			tb.Write(beforeCommitID)
		}

		style := ir.getSegmentStyle(*segment)
		if sr, ok := ir.op.(operations.SegmentRenderer); ok {
			rendered := sr.RenderSegment(style, segment, ir.row)
			if rendered != "" {
				tb.Write(rendered)
				continue
			}
		}
		tb.Styled(segment.Text, style)
	}

	// Add affected marker
	if line.Flags&parser.Revision == parser.Revision && ir.row.IsAffected {
		style := ir.dimmedStyle
		tb.Styled(" (affected by last operation)", style)
	}
}

// renderOperationLine renders an operation line with gutter
func (r *DisplayContextRenderer) renderOperationLine(
	dl *render.DisplayContext,
	rect cellbuf.Rectangle,
	gutter parser.GraphGutter,
	line string,
) {
	dl.AddFill(rect, ' ', lipgloss.NewStyle(), 0)
	tb := dl.Text(rect.Min.X, rect.Min.Y, 0)
	// Render gutter with text style (matching original behavior)
	for _, segment := range gutter.Segments {
		style := segment.Style.Inherit(r.textStyle)
		tb.Styled(segment.Text, style)
	}

	// Add line content
	tb.Write(line)
	tb.Done()
}

// renderGutter renders just the gutter portion (for embedded operations)
func (r *DisplayContextRenderer) renderGutter(gutter parser.GraphGutter) string {
	var result strings.Builder
	for _, segment := range gutter.Segments {
		style := segment.Style.Inherit(r.textStyle)
		result.WriteString(style.Render(segment.Text))
	}
	return result.String()
}

// GetScrollOffset returns the current scroll offset
func (r *DisplayContextRenderer) GetScrollOffset() int {
	return r.listRenderer.GetScrollOffset()
}

// SetScrollOffset sets the scroll offset
func (r *DisplayContextRenderer) SetScrollOffset(offset int) {
	r.listRenderer.SetScrollOffset(offset)
}

// GetFirstRowIndex returns the first visible row index.
func (r *DisplayContextRenderer) GetFirstRowIndex() int {
	return r.listRenderer.GetFirstRowIndex()
}

// GetLastRowIndex returns the last visible row index (inclusive).
func (r *DisplayContextRenderer) GetLastRowIndex() int {
	return r.listRenderer.GetLastRowIndex()
}
