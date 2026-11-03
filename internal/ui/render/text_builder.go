package render

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/layout"
)

type TextBuilder struct {
	dl        *DisplayContext
	segments  []textSegment
	x         int
	y         int
	z         int
	bounded   bool
	maxWidth  int
	maxHeight int
}

type textSegment struct {
	text    string
	style   lipgloss.Style
	onClick tea.Msg
}

const ellipsis = "…"

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

// TextIn returns a TextBuilder bounded by rect. Lines wider than rect are
// truncated with an ellipsis and lines below it are dropped.
func (dl *DisplayContext) TextIn(rect layout.Rectangle, z int) *TextBuilder {
	return &TextBuilder{
		dl:        dl,
		x:         rect.Min.X,
		y:         rect.Min.Y,
		z:         z,
		bounded:   true,
		maxWidth:  max(rect.Dx(), 0),
		maxHeight: max(rect.Dy(), 0),
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
		segRect := layout.Rect(tb.x+seg.x, tb.y+seg.y, seg.width, 1)
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

	lineStart := 0
	lineClipped := false

	for _, seg := range tb.segments {
		parts := strings.Split(seg.text, "\n")
		for i, part := range parts {
			if i > 0 {
				if col > maxWidth {
					maxWidth = col
				}
				row++
				col = 0
				lineStart = len(segments)
				lineClipped = false
			}

			if part == "" {
				continue
			}
			if tb.bounded && row >= tb.maxHeight {
				continue
			}

			rendered := seg.style.Render(part)
			width := StringWidth(rendered)
			if width == 0 {
				continue
			}

			if tb.bounded {
				available := tb.maxWidth - col
				if lineClipped {
					continue
				}
				if available <= 0 {
					// The previous segment filled the line exactly; mark the cut on it.
					if last := len(segments) - 1; last >= lineStart {
						// The padding forces Truncate to cut, keeping the ellipsis inside the style.
						segments[last].rendered = widthMethod.Truncate(segments[last].rendered+" ", segments[last].width, ellipsis)
					}
					lineClipped = true
					continue
				}
				if width > available {
					rendered = widthMethod.Truncate(rendered, available, ellipsis)
					width = StringWidth(rendered)
					lineClipped = true
				}
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
	if tb.bounded {
		height = min(height, tb.maxHeight)
	}
	return segments, maxWidth, height
}
