package common

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
)

type ListItem interface {
	list.DefaultItem
	ShortCut() string
}

type ListItemDelegate struct {
	ShowShortcuts bool
}

func (l ListItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var (
		title    string
		desc     string
		shortcut string
	)
	if item, ok := item.(ListItem); ok {
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
		title = title[:titleWidth-3] + "..."
	}

	if len(desc) > m.Width() {
		desc = desc[:m.Width()-3] + "..."
	}

	var (
		titleStyle    = DefaultPalette.Normal
		descStyle     = DefaultPalette.Dimmed
		shortcutStyle = DefaultPalette.Shortcut
	)

	highlightColor := lipgloss.AdaptiveColor{
		Light: config.Current.UI.HighlightLight,
		Dark:  config.Current.UI.HighlightDark,
	}

	if index == m.Index() {
		titleStyle = DefaultPalette.CompletionMatched.
			Bold(true).
			Background(highlightColor)
		descStyle = DefaultPalette.CompletionSelected.
			Background(highlightColor)
		shortcutStyle = shortcutStyle.Background(highlightColor)
	}

	titleStyle = titleStyle.Width(titleWidth)
	descStyle = descStyle.Width(m.Width())

	_, _ = fmt.Fprint(w, " ")
	if shortcut != "" {
		_, _ = fmt.Fprint(w, lipgloss.JoinHorizontal(0, shortcutStyle.Render(shortcut, ""), titleStyle.Render(title)))
	} else {
		_, _ = fmt.Fprint(w, titleStyle.Render(title))
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprint(w, " ")
	_, _ = fmt.Fprintf(w, descStyle.Render(desc))
	_, _ = fmt.Fprint(w, " ")
}

func (l ListItemDelegate) Height() int {
	return 1
}

func (l ListItemDelegate) Spacing() int {
	return 1
}

func (l ListItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}
