package render

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
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

type layoutSegment struct {
	x        int
	y        int
	width    int
	rendered string
	onClick  tea.Msg
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

func (tb *TextBuilder) NewLine() *TextBuilder {
	tb.segments = append(tb.segments, textSegment{text: "\n"})
	return tb
}

func (tb *TextBuilder) Space(count int) *TextBuilder {
	if count <= 0 {
		return tb
	}
	tb.segments = append(tb.segments, textSegment{text: strings.Repeat(" ", count)})
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

func (tb *TextBuilder) Measure() (int, int) {
	_, width, height := tb.layout()
	return width, height
}

func (tb *TextBuilder) Done() {
	segs, _, _ := tb.layout()
	for _, seg := range segs {
		segRect := cellbuf.Rect(tb.x+seg.x, tb.y+seg.y, seg.width, 1)
		tb.dl.AddDraw(segRect, seg.rendered, tb.z)
		if seg.onClick != nil {
			tb.dl.AddInteraction(segRect, seg.onClick, InteractionClick, tb.z)
		}
	}
}

func (tb *TextBuilder) layout() ([]layoutSegment, int, int) {
	var segments []layoutSegment
	maxWidth := 0
	row := 0
	col := 0
	hasContent := false

	for _, seg := range tb.segments {
		parts := strings.Split(seg.text, "\n")
		for i, part := range parts {
			if i > 0 {
				if col > maxWidth {
					maxWidth = col
				}
				row++
				col = 0
			}

			if part == "" {
				continue
			}

			rendered := seg.style.Render(part)
			width := lipgloss.Width(rendered)
			if width == 0 {
				continue
			}

			segments = append(segments, layoutSegment{
				x:        col,
				y:        row,
				width:    width,
				rendered: rendered,
				onClick:  seg.onClick,
			})
			col += width
			hasContent = true
			if col > maxWidth {
				maxWidth = col
			}
		}
	}

	if !hasContent {
		return nil, 0, 0
	}

	height := row + 1
	return segments, maxWidth, height
}
