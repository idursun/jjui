package oplog

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common/list"
)

type OpLogList struct {
	*list.List[*models.OperationLogItem]
	renderer      *list.ListRenderer[*models.OperationLogItem]
	selectedStyle lipgloss.Style
	textStyle     lipgloss.Style
}

func (o *OpLogList) RenderItem(w io.Writer, index int) {
	row := o.Items[index]
	isHighlighted := index == o.Cursor

	for _, rowLine := range row.Lines {
		lw := strings.Builder{}
		for _, segment := range rowLine.Segments {
			if isHighlighted {
				fmt.Fprint(&lw, segment.Style.Inherit(o.selectedStyle).Render(segment.Text))
			} else {
				fmt.Fprint(&lw, segment.Style.Inherit(o.textStyle).Render(segment.Text))
			}
		}
		line := lw.String()
		if isHighlighted {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(o.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(o.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(o.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(o.textStyle.GetBackground())))
		}
		fmt.Fprint(w, "\n")
	}
}

func (o *OpLogList) GetItemHeight(index int) int {
	return len(o.Items[index].Lines)
}
