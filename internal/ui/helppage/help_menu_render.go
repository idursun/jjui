package helppage

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m itemMenu) calculateMaxHeight() int {
	return max(
		m.leftList.getListHeight(),
		m.middleList.getListHeight(),
		m.rightList.getListHeight(),
	)
}

func (list menuColumn) getListHeight() int {
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
	padLine := func(content string) string {
		return lipgloss.Place(
			width, 1, lipgloss.Left, lipgloss.Top,
			content,
			lipgloss.WithWhitespaceBackground(h.styles.text.GetBackground()),
		)
	}

	for _, group := range column {
		for _, item := range group {
			lines = append(lines, padLine(item.display))
		}
		lines = append(lines, padLine(""))
	}

	for len(lines) < height {
		lines = append(lines, padLine(""))
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
