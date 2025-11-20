package evolog

import (
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common/list"
)

var _ list.IItemRenderer = (*itemRenderer)(nil)

type itemRenderer struct {
	row           parser.Row
	styleOverride lipgloss.Style
}

func (r itemRenderer) Render(w io.Writer, width int) {
	row := r.row
	for lineIndex := 0; lineIndex < len(row.Lines); lineIndex++ {
		segmentedLine := row.Lines[lineIndex]

		lw := strings.Builder{}
		for _, segment := range segmentedLine.Gutter.Segments {
			style := segment.Style
			fmt.Fprint(&lw, style.Render(segment.Text))
		}

		for _, segment := range segmentedLine.Segments {
			style := segment.Style.Inherit(r.styleOverride)
			fmt.Fprint(&lw, style.Render(segment.Text))
		}
		line := lw.String()
		fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceStyle(r.styleOverride)))
		fmt.Fprint(w, "\n")
	}
}

func (r itemRenderer) Height() int {
	return len(r.row.Lines)
}
