package oplog

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/ops"
)

var _ list.IItemRenderer = (*itemRenderer)(nil)

type itemRenderer struct {
	row   row
	style lipgloss.Style
}

func (i itemRenderer) Render(dl *ops.DisplayList, rect cellbuf.Rectangle, width int) {
	var sb strings.Builder
	row := i.row

	for _, rowLine := range row.Lines {
		lw := strings.Builder{}
		for _, segment := range rowLine.Segments {
			fmt.Fprint(&lw, segment.Style.Inherit(i.style).Render(segment.Text))
		}
		line := lw.String()
		fmt.Fprint(&sb, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(i.style.GetBackground())))
		fmt.Fprint(&sb, "\n")
	}

	content := sb.String()
	if content != "" {
		content = strings.TrimSuffix(content, "\n")
		dl.AddDraw(rect, content, 0)
	}
}

func (i itemRenderer) Height() int {
	return len(i.row.Lines)
}
