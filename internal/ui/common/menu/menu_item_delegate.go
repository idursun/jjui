package menu

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type MenuItem interface {
	list.DefaultItem
	ShortCut() string
}

type MenuItemDelegate struct {
	ShowShortcuts bool
	styles        styles
}

func (l MenuItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var (
		title    string
		desc     string
		shortcut string
	)
	if item, ok := item.(MenuItem); ok {
		title = item.Title()
		desc = item.Description()
		shortcut = item.ShortCut()
	} else {
		return
	}
	if m.Width() <= 0 {
		// short-circuit
		return
	}

	if !l.ShowShortcuts {
		shortcut = ""
	}

	titleWidth := m.Width()
	if shortcut != "" {
		titleWidth -= lipgloss.Width(shortcut) + 1
	}

	if len(title) > titleWidth {
		title = title[:titleWidth-1] + "…"
	}

	if len(desc) > m.Width() {
		desc = desc[:m.Width()-1] + "…"
	}

	var (
		titleStyle    = l.styles.text
		descStyle     = l.styles.dimmed
		shortcutStyle = l.styles.shortcut
	)

	if index == m.Index() {
		titleStyle = l.styles.selected
		descStyle = l.styles.selected
		shortcutStyle = shortcutStyle.Background(l.styles.selected.GetBackground())
	}

	titleLine := ""
	if shortcut != "" {
		titleLine = lipgloss.JoinHorizontal(0, shortcutStyle.PaddingLeft(1).Render(shortcut), titleStyle.PaddingLeft(1).Render(title))
	} else {
		titleLine = titleStyle.PaddingLeft(1).Render(title)
	}
	titleLine = lipgloss.PlaceHorizontal(m.Width()+2, 0, titleLine, lipgloss.WithWhitespaceStyle(titleStyle))

	descStyle = descStyle.PaddingLeft(1).PaddingRight(1).Width(m.Width() + 2)
	descLine := descStyle.Render(desc)
	descLine = lipgloss.PlaceHorizontal(m.Width()+2, 0, descLine, lipgloss.WithWhitespaceStyle(titleStyle))

	rendered := lipgloss.JoinVertical(lipgloss.Left, titleLine, descLine)
	_, _ = fmt.Fprint(w, rendered)
}

func (l MenuItemDelegate) Height() int {
	return 2
}

func (l MenuItemDelegate) Spacing() int {
	return 1
}

func (l MenuItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}
