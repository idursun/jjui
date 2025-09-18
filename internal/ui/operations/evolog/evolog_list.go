package evolog

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common/list"
)

var _ list.IItemRenderer = (*EvologList)(nil)
var _ list.IListProvider = (*EvologList)(nil)

type EvologList struct {
	*list.List[*models.RevisionItem]
	renderer      *list.ListRenderer[*models.RevisionItem]
	selectedStyle lipgloss.Style
	textStyle     lipgloss.Style
	dimmedStyle   lipgloss.Style
	commitIdStyle lipgloss.Style
	changeIdStyle lipgloss.Style
	markerStyle   lipgloss.Style
}

func (e *EvologList) CurrentItem() models.IItem {
	return e.Current()
}

func (e *EvologList) CheckedItems() []models.IItem {
	return nil
}

func (e *EvologList) RenderItem(w io.Writer, index int) {
	row := e.Items[index]
	isHighlighted := index == e.Cursor
	for lineIndex := 0; lineIndex < len(row.Lines); lineIndex++ {
		segmentedLine := row.Lines[lineIndex]

		lw := strings.Builder{}
		for _, segment := range segmentedLine.Gutter.Segments {
			style := segment.Style
			fmt.Fprint(&lw, style.Render(segment.Text))
		}

		for _, segment := range segmentedLine.Segments {
			style := segment.Style
			if isHighlighted {
				style = style.Inherit(e.selectedStyle)
			}
			fmt.Fprint(&lw, style.Render(segment.Text))
		}
		line := lw.String()
		if isHighlighted && segmentedLine.Flags&models.Highlightable == models.Highlightable {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(e.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(e.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(e.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(e.textStyle.GetBackground())))
		}
		fmt.Fprint(w, "\n")
	}
}

func (e *EvologList) GetItemHeight(index int) int {
	return len(e.Items[index].Lines)
}
