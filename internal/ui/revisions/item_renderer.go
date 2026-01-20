package revisions

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

type itemRenderer struct {
	row           parser.Row
	isHighlighted bool
	selectedStyle lipgloss.Style
	textStyle     lipgloss.Style
	dimmedStyle   lipgloss.Style
	matchedStyle  lipgloss.Style
	op            operations.Operation
	SearchText    string
	AceJumpPrefix *string
	isChecked     bool
}

// getSegmentStyleForLine returns the style for a segment, considering whether the line is highlightable.
// Only lines with the Highlightable flag should get the selected style when the row is selected.
func (ir itemRenderer) getSegmentStyleForLine(segment screen.Segment, lineIsHighlightable bool) lipgloss.Style {
	style := segment.Style
	if ir.isHighlighted && lineIsHighlightable {
		style = style.Inherit(ir.selectedStyle)
	} else {
		style = style.Inherit(ir.textStyle)
	}
	return style
}

// renderSegmentForLine renders a segment considering whether the line is highlightable.
func (ir itemRenderer) renderSegmentForLine(tb *render.TextBuilder, segment *screen.Segment, lineIsHighlightable bool) {
	baseStyle := ir.getSegmentStyleForLine(*segment, lineIsHighlightable)
	if ir.SearchText == "" {
		tb.Styled(segment.Text, baseStyle)
		return
	}

	lowerText := strings.ToLower(segment.Text)
	searchText := ir.SearchText
	if !strings.Contains(lowerText, searchText) {
		tb.Styled(segment.Text, baseStyle)
		return
	}

	matchStyle := baseStyle.Inherit(ir.matchedStyle)
	start := 0
	for {
		offset := strings.Index(lowerText[start:], searchText)
		if offset == -1 {
			if start < len(segment.Text) {
				tb.Styled(segment.Text[start:], baseStyle)
			}
			break
		}
		idx := start + offset
		if idx > start {
			tb.Styled(segment.Text[start:idx], baseStyle)
		}
		end := idx + len(searchText)
		if end > len(segment.Text) {
			end = len(segment.Text)
		}
		tb.Styled(segment.Text[idx:end], matchStyle)
		start = end
		if start >= len(segment.Text) {
			break
		}
	}
}
