package revisions

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/operations"
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

func (ir itemRenderer) getSegmentStyle(segment screen.Segment) lipgloss.Style {
	style := segment.Style
	if ir.isHighlighted {
		style = style.Inherit(ir.selectedStyle)
	} else {
		style = style.Inherit(ir.textStyle)
	}
	return style
}
