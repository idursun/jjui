package oplog

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common/list"
)

var _ list.IItemRenderer = (*itemRenderer)(nil)

type itemRenderer struct {
	row   row
	style lipgloss.Style
}

func (i itemRenderer) Render(w list.CursorWriter, width int) {
	row := i.row

	for _, rowLine := range row.Lines {
		lw := strings.Builder{}
		for _, segment := range rowLine.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(i.style).Render(segment.Text))
		}
		line := lw.String()
		fmt.Fprint(w, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(i.style.GetBackground())))
		fmt.Fprint(w, "\n")
	}
}

func (i itemRenderer) Height() int {
	return len(i.row.Lines)
}
