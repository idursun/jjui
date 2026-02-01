package details

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// renderFlatList renders files as a flat scrollable list
func (d *DetailsList) renderFlatList(dl *render.DisplayContext, viewRect layout.Box) {
	// Measure function - all items have height 1
	measure := func(index int) int {
		return 1
	}

	// Render function - renders each visible item
	renderItem := func(dl *render.DisplayContext, index int, rect cellbuf.Rectangle) {
		item := d.files[index]
		isSelected := index == d.cursor

		baseStyle := d.getStatusStyle(item.status)
		if isSelected {
			baseStyle = baseStyle.Bold(true).Background(d.styles.Selected.GetBackground())
		} else {
			baseStyle = baseStyle.Background(d.styles.Text.GetBackground())
		}
		background := lipgloss.NewStyle().Background(baseStyle.GetBackground())
		dl.AddFill(rect, ' ', background, 0)

		tb := dl.Text(rect.Min.X, rect.Min.Y, 0)
		d.renderItemContent(tb, item, isSelected, item.name, baseStyle)
		tb.Done()

		// Add highlight for selected item
		if isSelected {
			style := d.getStatusStyle(item.status).Bold(true).Background(d.styles.Selected.GetBackground())
			dl.AddHighlight(rect, style, 1)
		}
	}

	// Click message factory
	clickMsg := func(index int) render.ClickMessage {
		return FileClickedMsg{Index: index}
	}

	// Use the generic list renderer
	d.listRenderer.Render(
		dl,
		viewRect,
		len(d.files),
		d.cursor,
		d.ensureCursorView,
		measure,
		renderItem,
		clickMsg,
	)
	d.listRenderer.RegisterScroll(dl, viewRect)
}

// findFileIndex finds the index of a file by name in the flat list
func (d *DetailsList) findFileIndex(fileName string) int {
	for i, file := range d.files {
		if file.fileName == fileName {
			return i
		}
	}
	return -1
}
