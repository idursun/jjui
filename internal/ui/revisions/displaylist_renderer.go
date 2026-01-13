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
	listRenderer  *render.ListRenderer
	selections    map[string]bool
	textStyle     lipgloss.Style
	dimmedStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	matchedStyle  lipgloss.Style
}

// NewDisplayListRenderer creates a new DisplayList-based renderer
func NewDisplayListRenderer(textStyle, dimmedStyle, selectedStyle, matchedStyle lipgloss.Style) *DisplayListRenderer {
	return &DisplayListRenderer{
		listRenderer:  render.NewListRenderer(ViewportScrollMsg{}),
		textStyle:     textStyle,
		dimmedStyle:   dimmedStyle,
		selectedStyle: selectedStyle,
		matchedStyle:  matchedStyle,
	}
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

	// Measure function - calculates height for each item
	measure := func(index int) int {
		item := items[index]
		isSelected := index == cursor
		return r.calculateItemHeight(item, isSelected, operation)
	}

	// Screen offset for interactions (absolute screen position)
	screenOffset := cellbuf.Pos(viewRect.R.Min.X, viewRect.R.Min.Y)

	// Render function - renders each visible item
	renderItem := func(dl *render.DisplayList, index int, rect cellbuf.Rectangle) {
		item := items[index]
		isSelected := index == cursor

		// Render the item content
		r.renderItemToDisplayList(dl, item, rect, isSelected, operation, screenOffset)

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
}

// addHighlights adds highlight effects for lines with Highlightable flag
func (r *DisplayListRenderer) addHighlights(
	dl *render.DisplayList,
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
	screenOffset cellbuf.Position,
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

				content := r.renderOperationLine(extended, line, width)
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
	descriptionRendered := false

	for i := 0; i < len(item.Lines); i++ {
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

				content := r.renderOperationLine(line.Gutter, overlayLine, width)
				lineRect := cellbuf.Rect(rect.Min.X, y, rect.Dx(), 1)
				dl.AddDraw(lineRect, content, 0)
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

		content := ir.renderLineToString(line, width)
		lineRect := cellbuf.Rect(rect.Min.X, y, rect.Dx(), 1)
		dl.AddDraw(lineRect, content, 0)
		y++
	}

	// Handle operation rendering for after section
	if isSelected && operation != nil && !item.Commit.IsRoot() {
		// Check if operation supports DisplayList rendering
		if dlRenderer, ok := operation.(operations.DisplayListRenderer); ok && dlRenderer.SupportsDisplayList(operations.RenderPositionAfter) {
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
			height := dlRenderer.RenderToDisplayList(dl, item.Commit, operations.RenderPositionAfter, contentRect, contentScreenOffset)

			// Render gutters for each line
			for i := 0; i < height; i++ {
				gutterContent := r.renderGutter(extended)
				gutterRect := cellbuf.Rect(rect.Min.X, y+i, gutterWidth, 1)
				dl.AddDraw(gutterRect, gutterContent, 0)
			}
		} else {
			// Fall back to string-based rendering
			after := operation.Render(item.Commit, operations.RenderPositionAfter)
			if after != "" {
				lines := strings.Split(after, "\n")
				extended := item.Extend()

				for _, line := range lines {
					if y >= rect.Max.Y {
						break
					}

					content := r.renderOperationLine(extended, line, width)
					lineRect := cellbuf.Rect(rect.Min.X, y, rect.Dx(), 1)
					dl.AddDraw(lineRect, content, 0)
					y++
				}
			}
		}
	}
}

// renderLineToString renders a line to a string (helper for itemRenderer)
func (ir *itemRenderer) renderLineToString(line *parser.GraphRowLine, width int) string {
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
func (r *DisplayListRenderer) renderOperationLine(gutter parser.GraphGutter, line string, width int) string {
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

// renderGutter renders just the gutter portion (for embedded operations)
func (r *DisplayListRenderer) renderGutter(gutter parser.GraphGutter) string {
	var result strings.Builder
	for _, segment := range gutter.Segments {
		style := segment.Style.Inherit(r.textStyle)
		result.WriteString(style.Render(segment.Text))
	}
	return result.String()
}

// GetScrollOffset returns the current scroll offset
func (r *DisplayListRenderer) GetScrollOffset() int {
	return r.listRenderer.GetScrollOffset()
}

// SetScrollOffset sets the scroll offset
func (r *DisplayListRenderer) SetScrollOffset(offset int) {
	r.listRenderer.SetScrollOffset(offset)
}
