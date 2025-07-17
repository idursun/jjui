package oplog

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"io"
	"strings"
)

type iterator struct {
	Palette             common.Palette
	HighlightBackground lipgloss.AdaptiveColor
	Width               int
	Rows                []row
	isHighlighted       bool
	current             int
	Cursor              int
}

func newIterator(rows []row, cursor int, width int) *iterator {
	return &iterator{
		Palette:             common.DefaultPalette,
		HighlightBackground: lipgloss.AdaptiveColor{Light: config.Current.UI.HighlightLight, Dark: config.Current.UI.HighlightDark},
		Width:               width,
		Rows:                rows,
		isHighlighted:       false,
		current:             -1,
		Cursor:              cursor,
	}
}

func (o *iterator) IsHighlighted() bool {
	return o.current == o.Cursor
}

func (o *iterator) Render(r io.Writer) {
	row := o.Rows[o.current]
	renderer := o

	for _, rowLine := range row.Lines {
		lw := strings.Builder{}
		for _, segment := range rowLine.Segments {
			style := segment.Style
			if o.isHighlighted {
				style = style.Background(o.HighlightBackground)
			}
			fmt.Fprint(&lw, style.Render(segment.Text))
		}
		line := lw.String()
		fmt.Fprint(r, line)
		if o.isHighlighted {
			lineWidth := lipgloss.Width(line)
			gap := renderer.Width - lineWidth
			if gap > 0 {
				fmt.Fprint(r, lipgloss.NewStyle().Background(o.HighlightBackground).Render(strings.Repeat(" ", gap)))
			}
		}
		fmt.Fprint(r, "\n")
	}
}

func (o *iterator) RowHeight() int {
	return len(o.Rows[o.current].Lines)
}

func (o *iterator) Next() bool {
	o.current++
	if o.current >= len(o.Rows) {
		return false
	}
	o.isHighlighted = o.current == o.Cursor
	return true
}

func (o *iterator) Len() int {
	return len(o.Rows)
}
