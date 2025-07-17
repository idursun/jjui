package oplog

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"io"
	"strings"
)

type iterator struct {
	Width         int
	Rows          []row
	isHighlighted bool
	current       int
	Cursor        int
	SelectedStyle lipgloss.Style
	TextStyle     lipgloss.Style
}

func newIterator(rows []row, cursor int, width int) *iterator {
	return &iterator{
		Width:         width,
		Rows:          rows,
		isHighlighted: false,
		current:       -1,
		Cursor:        cursor,
		SelectedStyle: common.DefaultPalette.Get("oplog selected").Inline(true),
		TextStyle:     common.DefaultPalette.Get("oplog text").Inline(true),
	}
}

func (o *iterator) IsHighlighted() bool {
	return o.current == o.Cursor
}

func (o *iterator) Render(r io.Writer) {
	row := o.Rows[o.current]

	for _, rowLine := range row.Lines {
		lw := strings.Builder{}
		for _, segment := range rowLine.Segments {
			if o.isHighlighted {
				fmt.Fprint(&lw, segment.Style.Inherit(o.SelectedStyle).Render(segment.Text))
			} else {
				fmt.Fprint(&lw, segment.Style.Inherit(o.TextStyle).Render(segment.Text))
			}
		}
		line := lw.String()
		if o.isHighlighted {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(o.Width, 0, line, lipgloss.WithWhitespaceBackground(o.SelectedStyle.GetBackground())))
		} else {
			fmt.Fprint(r, lipgloss.PlaceHorizontal(o.Width, 0, line, lipgloss.WithWhitespaceBackground(o.TextStyle.GetBackground())))
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
