package oplog

import (
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/common/list"
)

var _ list.IItemRenderer = (*itemRenderer)(nil)

type itemRenderer struct {
	row   row
	style lipgloss.Style
}

func (i itemRenderer) Render(w io.Writer, width int) {
	row := i.row

	for _, rowLine := range row.Lines {
		lw := strings.Builder{}
		for _, segment := range rowLine.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(i.style).Render(segment.Text))
		}
		line := lw.String()
		fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceStyle(i.style)))
		fmt.Fprint(w, "\n")
	}
}

func (i itemRenderer) Height() int {
	return len(i.row.Lines)
}
