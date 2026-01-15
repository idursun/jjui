package render

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/mattn/go-runewidth"
)

// TextBuilder provides a fluent API for composing text with styled and
// interactive segments. It accumulates segments and flushes them to
// the DisplayList in a single operation.
type TextBuilder struct {
	dl       *DisplayList
	segments []textSegment
	x        int // current x position
	y        int // y position (single line)
	z        int // z-index for all operations
}

type textSegment struct {
	text    string
	style   lipgloss.Style
	onClick tea.Msg
}

// Text creates a new TextBuilder starting at the given position.
// The rect's X and Y define the starting position; width/height are ignored.
func (dl *DisplayList) Text(x, y, z int) *TextBuilder {
	return &TextBuilder{
		dl: dl,
		x:  x,
		y:  y,
		z:  z,
	}
}

// Write adds plain unstyled text.
func (tb *TextBuilder) Write(text string) *TextBuilder {
	tb.segments = append(tb.segments, textSegment{text: text})
	return tb
}

// Styled adds text with a lipgloss style applied.
func (tb *TextBuilder) Styled(text string, style lipgloss.Style) *TextBuilder {
	tb.segments = append(tb.segments, textSegment{text: text, style: style})
	return tb
}

// Clickable adds text that responds to mouse clicks by sending the given message.
func (tb *TextBuilder) Clickable(text string, style lipgloss.Style, onClick tea.Msg) *TextBuilder {
	tb.segments = append(tb.segments, textSegment{
		text:    text,
		style:   style,
		onClick: onClick,
	})
	return tb
}

// Done flushes all accumulated segments to the DisplayList.
// It renders each segment's text and registers any click interactions.
func (tb *TextBuilder) Done() {
	x := tb.x

	for _, seg := range tb.segments {
		width := runewidth.StringWidth(seg.text)
		if width == 0 {
			continue
		}

		segRect := cellbuf.Rect(x, tb.y, width, 1)

		// Render the text with styling (empty style returns text unchanged)
		tb.dl.AddDraw(segRect, seg.style.Render(seg.text), tb.z)

		// Register click interaction if provided
		if seg.onClick != nil {
			tb.dl.AddInteraction(segRect, seg.onClick, InteractionClick, tb.z)
		}

		x += width
	}
}
