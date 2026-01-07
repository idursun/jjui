package evolog

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/ops"
)

var _ list.IItemRenderer = (*itemRenderer)(nil)

type itemRenderer struct {
	row           parser.Row
	styleOverride lipgloss.Style
}

func (r itemRenderer) Render(dl *ops.DisplayList, rect cellbuf.Rectangle, width int) {
	var sb strings.Builder
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
		fmt.Fprint(&sb, lipgloss.PlaceHorizontal(width, 0, line, lipgloss.WithWhitespaceBackground(r.styleOverride.GetBackground())))
		fmt.Fprint(&sb, "\n")
	}

	content := sb.String()
	if content != "" {
		content = strings.TrimSuffix(content, "\n")
		dl.AddDraw(rect, content, 0)
	}
}

func (r itemRenderer) Height() int {
	return len(r.row.Lines)
}
