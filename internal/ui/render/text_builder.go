package render

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/mattn/go-runewidth"
)

type TextBuilder struct {
	dl       *DisplayContext
	segments []textSegment
	x        int
	y        int
	z        int
}

type textSegment struct {
	text    string
	style   lipgloss.Style
	onClick tea.Msg
}

func (dl *DisplayContext) Text(x, y, z int) *TextBuilder {
	return &TextBuilder{
		dl: dl,
		x:  x,
		y:  y,
		z:  z,
	}
}

func (tb *TextBuilder) Write(text string) *TextBuilder {
	tb.segments = append(tb.segments, textSegment{text: text})
	return tb
}

func (tb *TextBuilder) Styled(text string, style lipgloss.Style) *TextBuilder {
	tb.segments = append(tb.segments, textSegment{text: text, style: style})
	return tb
}

func (tb *TextBuilder) Clickable(text string, style lipgloss.Style, onClick tea.Msg) *TextBuilder {
	tb.segments = append(tb.segments, textSegment{
		text:    text,
		style:   style,
		onClick: onClick,
	})
	return tb
}

func (tb *TextBuilder) Done() {
	x := tb.x

	for _, seg := range tb.segments {
		width := runewidth.StringWidth(seg.text)
		if width == 0 {
			continue
		}

		segRect := cellbuf.Rect(x, tb.y, width, 1)

		tb.dl.AddDraw(segRect, seg.style.Render(seg.text), tb.z)

		if seg.onClick != nil {
			tb.dl.AddInteraction(segRect, seg.onClick, InteractionClick, tb.z)
		}

		x += width
	}
}
