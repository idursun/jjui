package menu

import "github.com/charmbracelet/lipgloss"

type Item interface {
	Title() string
	Description() string
	FilterValue() string
	ShortCut() string
}

func renderMenuItem(width int, styles styles, showShortcuts bool, cursor int, index int, item Item) string {
	var (
		title    string
		desc     string
		shortcut string
	)
	title = item.Title()
	desc = item.Description()
	shortcut = item.ShortCut()
	if width <= 0 {
		// short-circuit
		return ""
	}

	if !showShortcuts {
		shortcut = ""
	}

	titleWidth := width
	if shortcut != "" {
		titleWidth -= lipgloss.Width(shortcut) + 1
	}

	if titleWidth > 0 && len(title) > titleWidth {
		title = title[:titleWidth-1] + "…"
	}

	if len(desc) > width {
		desc = desc[:width-1] + "…"
	}

	var (
		titleStyle    = styles.text
		descStyle     = styles.dimmed
		shortcutStyle = styles.shortcut
	)

	if index == cursor {
		titleStyle = styles.selected
		descStyle = styles.selected
		shortcutStyle = shortcutStyle.Background(styles.selected.GetBackground())
	}

	titleLine := ""
	if shortcut != "" {
		titleLine = lipgloss.JoinHorizontal(0, shortcutStyle.PaddingLeft(1).Render(shortcut), titleStyle.PaddingLeft(1).Render(title))
	} else {
		titleLine = titleStyle.PaddingLeft(1).Render(title)
	}
	titleLine = lipgloss.PlaceHorizontal(width+2, 0, titleLine, lipgloss.WithWhitespaceBackground(titleStyle.GetBackground()))

	descStyle = descStyle.PaddingLeft(1).PaddingRight(1).Width(width + 2)
	descLine := descStyle.Render(desc)
	descLine = lipgloss.PlaceHorizontal(width+2, 0, descLine, lipgloss.WithWhitespaceBackground(titleStyle.GetBackground()))

	return lipgloss.JoinVertical(lipgloss.Left, titleLine, descLine)
}
