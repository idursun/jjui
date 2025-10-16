package helppage

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (h *Model) calculateMaxHeight() int {
	return max(
		getListHeight(h.defaultMenu.leftList),
		getListHeight(h.defaultMenu.middleList),
		getListHeight(h.defaultMenu.rightList),
	)
}

func getListHeight(list menuColumn) int {
	height := 0
	for _, group := range list {
		height += len(group)
		height++ // spacing between groups
	}
	return height
}

func (h *Model) renderColumn(column menuColumn) string {
	// NOTE: read from defaultMenu so layout won't glitch while filtering menu
	width := h.defaultMenu.width / 3
	height := h.defaultMenu.height
	var lines []string
	formatLine := func(content string) string {
		return lipgloss.Place(
			width, 1, lipgloss.Left, lipgloss.Top,
			content,
			lipgloss.WithWhitespaceBackground(h.styles.text.GetBackground()),
		)
	}

	for _, group := range column {
		for _, item := range group {
			lines = append(lines, formatLine(item.display))
		}
		lines = append(lines, formatLine(""))
	}

	for len(lines) < height {
		lines = append(lines, formatLine(""))
	}

	return strings.Join(lines, "\n")
}

func (h *Model) renderMenu() string {
	if h.searchQuery.Value() == "" {
		h.filteredMenu = h.defaultMenu
	}

	left := h.renderColumn(h.filteredMenu.leftList)
	middle := h.renderColumn(h.filteredMenu.middleList)
	right := h.renderColumn(h.filteredMenu.rightList)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, middle, right)
}
