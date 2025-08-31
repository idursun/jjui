package oplog

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type iterator struct {
	context       *context.OplogContext
	Width         int
	isHighlighted bool
	current       int
	SelectedStyle lipgloss.Style
	TextStyle     lipgloss.Style
}

func newIterator(context *context.OplogContext, width int) *iterator {
	return &iterator{
		context:       context,
		Width:         width,
		isHighlighted: false,
		current:       -1,
		SelectedStyle: common.DefaultPalette.Get("oplog selected").Inline(true),
		TextStyle:     common.DefaultPalette.Get("oplog text").Inline(true),
	}
}

func (o *iterator) IsHighlighted() bool {
	return o.current == o.context.Cursor
}

func (o *iterator) Render(r io.Writer) {
	row := o.context.Items[o.current]

	for _, segments := range row.Lines {
		lw := strings.Builder{}
		for _, segment := range segments {
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
	return len(o.context.Items[o.current].Lines)
}

func (o *iterator) Next() bool {
	o.current++
	if o.current >= len(o.context.Items) {
		return false
	}
	o.isHighlighted = o.current == o.context.Cursor
	return true
}

func (o *iterator) Len() int {
	return len(o.context.Items)
}
