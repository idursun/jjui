package common

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
)

type FilterableList struct {
	List          list.Model
	Items         []list.Item
	Filter        string
	KeyMap        config.KeyMappings[key.Binding]
	Width         int
	Height        int
	FilterMatches func(item list.Item, filter string) bool
	Title         string
}

type FilterMatchFunc func(list.Item, string) bool

func DefaultFilterMatch(item list.Item, filter string) bool {
	return true
}

func NewFilterableList(items []list.Item, width int, height int, keyMap config.KeyMappings[key.Binding]) FilterableList {
	l := list.New(items, ListItemDelegate{}, width, height)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(true)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	l.Styles.NoItems = DefaultPalette.Dimmed

	m := FilterableList{
		List:          l,
		Items:         items,
		KeyMap:        keyMap,
		Width:         width,
		Height:        height,
		FilterMatches: DefaultFilterMatch,
	}

	return m
}

func (m *FilterableList) SetWidth(w int) {
	m.Width = w
	m.List.SetWidth(w)
}

func (m *FilterableList) SetHeight(h int) {
	maxHeight, minHeight := 30, 10
	m.Height = max(min(maxHeight, h-4), minHeight)
	m.List.SetHeight(m.Height - 6)
}

func (m *FilterableList) ShowShortcuts(show bool) {
	m.List.SetDelegate(ListItemDelegate{ShowShortcuts: show})
}

func (m *FilterableList) Filtered(filter string) tea.Cmd {
	m.Filter = filter
	if m.Filter == "" {
		m.List.SetDelegate(ListItemDelegate{ShowShortcuts: false})
		return m.List.SetItems(m.Items)
	}

	m.List.SetDelegate(ListItemDelegate{ShowShortcuts: true})
	var filtered []list.Item
	for _, i := range m.Items {
		if m.FilterMatches(i, m.Filter) {
			filtered = append(filtered, i)
		}
	}
	m.List.ResetSelected()
	return m.List.SetItems(filtered)
}

func (m *FilterableList) RenderFilterView() string {
	filterStyle := DefaultPalette.Shortcut.PaddingLeft(1)
	filterValueStyle := DefaultPalette.Normal.Bold(true)

	filterView := lipgloss.JoinHorizontal(0, filterStyle.Render("Showing "), filterValueStyle.Render("all"))
	if m.Filter != "" {
		filterView = lipgloss.JoinHorizontal(0, filterStyle.Render("Showing only "), filterValueStyle.Render(m.Filter))
	}
	return filterView
}

func (m *FilterableList) RenderHelpView(helpKeys []key.Binding) string {
	if m.List.SettingFilter() {
		return ""
	}

	bindings := make([]string, 0, len(helpKeys)+1)
	for _, k := range helpKeys {
		if renderedKey := RenderKey(k); renderedKey != "" {
			bindings = append(bindings, renderedKey)
		}
	}

	if m.List.IsFiltered() {
		bindings = append(bindings, RenderKey(m.KeyMap.Cancel))
	} else {
		bindings = append(bindings, RenderKey(m.List.KeyMap.Filter))
	}

	return " " + lipgloss.JoinHorizontal(0, bindings...)
}

func RenderKey(k key.Binding) string {
	if !k.Enabled() {
		return ""
	}
	return lipgloss.JoinHorizontal(0, DefaultPalette.Shortcut.Render(k.Help().Key, ""), DefaultPalette.Dimmed.Render(k.Help().Desc, ""))
}

func (m *FilterableList) View(helpKeys []key.Binding) string {
	titleView := m.List.Styles.Title.Render(m.Title)
	filterView := m.RenderFilterView()
	listView := m.List.View()
	helpView := m.RenderHelpView(helpKeys)
	content := lipgloss.JoinVertical(0, titleView, "", filterView, listView, "", helpView)
	content = lipgloss.Place(m.Width, m.Height, 0, 0, content)
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Render(content)
}
