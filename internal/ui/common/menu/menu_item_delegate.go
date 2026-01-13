package menu

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	rendered := renderMenuItem(m, l.styles, l.ShowShortcuts, index, item)
	if rendered == "" {
		return
	}
	_, _ = fmt.Fprint(w, rendered)
}

func renderMenuItem(m list.Model, styles styles, showShortcuts bool, index int, item list.Item) string {
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
		return ""
	}
	if m.Width() <= 0 {
		// short-circuit
		return ""
	}

	if !showShortcuts {
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
		titleStyle    = styles.text
		descStyle     = styles.dimmed
		shortcutStyle = styles.shortcut
	)

	if index == m.Index() {
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
	titleLine = lipgloss.PlaceHorizontal(m.Width()+2, 0, titleLine, lipgloss.WithWhitespaceBackground(titleStyle.GetBackground()))

	descStyle = descStyle.PaddingLeft(1).PaddingRight(1).Width(m.Width() + 2)
	descLine := descStyle.Render(desc)
	descLine = lipgloss.PlaceHorizontal(m.Width()+2, 0, descLine, lipgloss.WithWhitespaceBackground(titleStyle.GetBackground()))

	return lipgloss.JoinVertical(lipgloss.Left, titleLine, descLine)
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
