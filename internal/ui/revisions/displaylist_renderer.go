package revisions

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

// DisplayListRenderer renders the revisions list using the DisplayList approach
type DisplayListRenderer struct {
	viewport      render.Viewport
	selections    map[string]bool
	textStyle     lipgloss.Style
	dimmedStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	matchedStyle  lipgloss.Style
}

// NewDisplayListRenderer creates a new DisplayList-based renderer
func NewDisplayListRenderer(textStyle, dimmedStyle, selectedStyle, matchedStyle lipgloss.Style) *DisplayListRenderer {
	return &DisplayListRenderer{
		textStyle:     textStyle,
		dimmedStyle:   dimmedStyle,
		selectedStyle: selectedStyle,
		matchedStyle:  matchedStyle,
		viewport: render.Viewport{
			StartLine: 0,
		},
	}
}

// SetViewport updates the viewport for scrolling
func (r *DisplayListRenderer) SetViewport(viewport render.Viewport) {
	r.viewport = viewport
}

// SetSelections sets the selected revisions for rendering checkboxes
func (r *DisplayListRenderer) SetSelections(selections map[string]bool) {
	r.selections = selections
}

// Render renders the revisions list to a DisplayList
func (r *DisplayListRenderer) Render(
	dl *render.DisplayList,
	items []parser.Row,
	cursor int,
	viewRect layout.Box,
	operation operations.Operation,
	ensureCursorVisible bool,
) {
	if len(items) == 0 {
		return
	}

	// Update viewport with current view rectangle
	r.viewport.ViewRect = layout.Box{
		R: cellbuf.Rect(0, 0, viewRect.R.Dx(), viewRect.R.Dy()),
	}

	// Ensure the cursor is visible by adjusting StartLine (only if requested)
	if ensureCursorVisible {
		r.ensureCursorVisible(items, cursor, operation)
	}

	// Use list_layout.LayoutAll to determine visible items
	measure := func(req render.MeasureRequest) render.MeasureResult {
		if req.Index >= len(items) {
			return render.MeasureResult{DesiredLine: 0, MinLine: 0}
		}

		item := items[req.Index]
		isSelected := req.Index == cursor
		height := r.calculateItemHeight(item, isSelected, operation)

		return render.MeasureResult{
			DesiredLine: height,
			MinLine:     height,
		}
	}

	spans, _ := render.LayoutAll(r.viewport, len(items), measure)

	// Render each visible item
	for _, span := range spans {
		if span.Index >= len(items) {
			continue
		}

		item := items[span.Index]
		isSelected := span.Index == cursor

		r.renderItemToDisplayList(
			dl,
			item,
			span.Rect,
			isSelected,
			operation,
			span.LineOffset,
			span.LineCount,
		)
	}

	// Add selection effect on top
	if cursor >= 0 && cursor < len(items) {
		for _, span := range spans {
			if span.Index == cursor {
				// Use highlight effect for selection highlight
				dl.AddHighlight(span.Rect, r.selectedStyle, 1000)
				break
			}
		}
	}
}

// calculateItemHeight calculates the height of an item in lines
func (r *DisplayListRenderer) calculateItemHeight(
	item parser.Row,
	isSelected bool,
	operation operations.Operation,
) int {
	// Base height from the item's lines
	height := len(item.Lines)

	// Add operation height if item is selected and operation exists
	if isSelected && operation != nil {
		// Count lines in before section
		before := operation.Render(item.Commit, operations.RenderPositionBefore)
		if before != "" {
			height += strings.Count(before, "\n") + 1
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
		after := operation.Render(item.Commit, operations.RenderPositionAfter)
		if after != "" {
			height += strings.Count(after, "\n") + 1
		}
	}

	return height
}

// renderItemToDisplayList renders a single item to the DisplayList
func (r *DisplayListRenderer) renderItemToDisplayList(
	dl *render.DisplayList,
	item parser.Row,
	rect cellbuf.Rectangle,
	isSelected bool,
	operation operations.Operation,
	lineOffset int,
	lineCount int,
) {
	y := rect.Min.Y
	width := rect.Dx()

	// Create an item renderer for this item
	ir := itemRenderer{
		row:           item,
		isHighlighted: isSelected,
		op:            operation,
		inLane:        true, // No tracer support yet, so everything is in lane
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

				content := r.renderOperationLine(extended, line, width, isSelected)
				lineRect := cellbuf.Rect(rect.Min.X, y, rect.Dx(), 1)
				dl.AddDraw(lineRect, content, 0)
				y++
			}
		}
	}

	// Handle main content and description overlay
	descriptionOverlay := ""
	if isSelected && operation != nil {
		descriptionOverlay = operation.Render(item.Commit, operations.RenderOverDescription)
	}

	// Render main lines
	linesPrinted := 0
	descriptionRendered := false

	for i := 0; i < len(item.Lines); i++ {
		if linesPrinted >= lineCount {
			break
		}

		line := item.Lines[i]

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

				content := r.renderOperationLine(line.Gutter, overlayLine, width, isSelected)
				lineRect := cellbuf.Rect(rect.Min.X, y, rect.Dx(), 1)
				dl.AddDraw(lineRect, content, 0)
				y++
				linesPrinted++
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

		content := ir.renderLineToString(line, i, width)
		lineRect := cellbuf.Rect(rect.Min.X, y, rect.Dx(), 1)
		dl.AddDraw(lineRect, content, 0)
		y++
		linesPrinted++
	}

	// Handle operation rendering for after section
	if isSelected && operation != nil && !item.Commit.IsRoot() {
		after := operation.Render(item.Commit, operations.RenderPositionAfter)
		if after != "" {
			lines := strings.Split(after, "\n")
			extended := item.Extend()

			for _, line := range lines {
				if y >= rect.Max.Y {
					break
				}

				content := r.renderOperationLine(extended, line, width, isSelected)
				lineRect := cellbuf.Rect(rect.Min.X, y, rect.Dx(), 1)
				dl.AddDraw(lineRect, content, 0)
				y++
			}
		}
	}
}

// renderLineToString renders a line to a string (helper for itemRenderer)
func (ir *itemRenderer) renderLineToString(line *parser.GraphRowLine, lineIndex int, width int) string {
	var result strings.Builder

	// Render gutter (no tracer support for now)
	for _, segment := range line.Gutter.Segments {
		style := segment.Style.Inherit(ir.textStyle)
		result.WriteString(style.Render(segment.Text))
	}

	// Add checkbox and operation content before ChangeID
	if line.Flags&parser.Revision == parser.Revision {
		if ir.isChecked {
			result.WriteString(ir.selectedStyle.Render("✓ "))
		}
		beforeChangeID := ir.op.Render(ir.row.Commit, operations.RenderBeforeChangeId)
		if beforeChangeID != "" {
			result.WriteString(beforeChangeID)
		}
	}

	// Render segments
	beforeCommitID := ""
	if ir.op != nil {
		beforeCommitID = ir.op.Render(ir.row.Commit, operations.RenderBeforeCommitId)
	}

	for _, segment := range line.Segments {
		if beforeCommitID != "" && segment.Text == ir.row.Commit.CommitId {
			result.WriteString(beforeCommitID)
		}

		style := ir.getSegmentStyle(*segment)
		result.WriteString(style.Render(segment.Text))
	}

	// Add affected marker
	if line.Flags&parser.Revision == parser.Revision && ir.row.IsAffected {
		style := ir.dimmedStyle
		result.WriteString(style.Render(" (affected by last operation)"))
	}

	// Pad to width
	content := result.String()
	if lipgloss.Width(content) < width {
		content = lipgloss.PlaceHorizontal(width, lipgloss.Left, content)
	}

	return content
}

// renderOperationLine renders an operation line with gutter
func (r *DisplayListRenderer) renderOperationLine(gutter parser.GraphGutter, line string, width int, isSelected bool) string {
	var result strings.Builder

	// Render gutter with text style (matching original behavior)
	for _, segment := range gutter.Segments {
		style := segment.Style.Inherit(r.textStyle)
		result.WriteString(style.Render(segment.Text))
	}

	// Add line content
	result.WriteString(line)

	// Pad to width
	content := result.String()
	if lipgloss.Width(content) < width {
		content = lipgloss.PlaceHorizontal(width, lipgloss.Left, content)
	}

	return content
}

// GetScrollOffset returns the current scroll offset
func (r *DisplayListRenderer) GetScrollOffset() int {
	return r.viewport.StartLine
}

// SetScrollOffset sets the scroll offset
func (r *DisplayListRenderer) SetScrollOffset(offset int) {
	r.viewport.StartLine = offset
}

// ensureCursorVisible adjusts the viewport StartLine only when cursor goes outside viewport
func (r *DisplayListRenderer) ensureCursorVisible(
	items []parser.Row,
	cursor int,
	operation operations.Operation,
) {
	if cursor < 0 || cursor >= len(items) {
		return
	}

	viewportHeight := r.viewport.ViewRect.R.Dy()
	if viewportHeight <= 0 {
		return
	}

	// Calculate the line position where the cursor item starts
	cursorStart := 0
	for i := 0; i < cursor && i < len(items); i++ {
		cursorStart += r.calculateItemHeight(items[i], false, operation)
	}

	// Calculate the height of the cursor item
	cursorHeight := r.calculateItemHeight(items[cursor], true, operation)
	cursorEnd := cursorStart + cursorHeight

	start := r.viewport.StartLine
	if start < 0 {
		start = 0
	}

	viewportEnd := start + viewportHeight

	// Only adjust if cursor is outside the current viewport
	// If cursor item starts before viewport top, scroll up to show it
	if cursorStart < start {
		r.viewport.StartLine = cursorStart
	} else if cursorEnd > viewportEnd {
		// If cursor item ends after viewport bottom, scroll down to show it
		r.viewport.StartLine = cursorEnd - viewportHeight
		if r.viewport.StartLine < 0 {
			r.viewport.StartLine = 0
		}
	}
	// Otherwise, cursor is already visible - don't adjust viewport
}
