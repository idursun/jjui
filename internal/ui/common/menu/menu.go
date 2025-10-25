package menu

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
)

type Menu struct {
	List          list.Model
	Items         []list.Item
	Filter        string
	KeyMap        config.KeyMappings[key.Binding]
	width         int
	height        int
	FilterMatches func(item list.Item, filter string) bool
	Title         string
	Subtitle      string
	styles        styles
}

type styles struct {
	title    lipgloss.Style
	subtitle lipgloss.Style
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
	selected lipgloss.Style
	matched  lipgloss.Style
	text     lipgloss.Style
	border   lipgloss.Style
}

type FilterMatchFunc func(list.Item, string) bool

func DefaultFilterMatch(item list.Item, filter string) bool {
	return true
}

type Option func(menu *Menu)

func WithStylePrefix(prefix string) Option {
	return func(menu *Menu) {
		menu.styles = createStyles(prefix)
	}
}

func createStyles(prefix string) styles {
	if prefix != "" {
		prefix += " "
	}
	return styles{
		title:    common.DefaultPalette.Get(prefix+"menu title").Padding(0, 1, 0, 1),
		subtitle: common.DefaultPalette.Get(prefix+"menu subtitle").Padding(1, 0, 0, 1),
		selected: common.DefaultPalette.Get(prefix + "menu selected"),
		matched:  common.DefaultPalette.Get(prefix + "menu matched"),
		dimmed:   common.DefaultPalette.Get(prefix + "menu dimmed"),
		shortcut: common.DefaultPalette.Get(prefix + "menu shortcut"),
		text:     common.DefaultPalette.Get(prefix + "menu text"),
		border:   common.DefaultPalette.GetBorder(prefix+"menu border", lipgloss.NormalBorder()),
	}
}

func NewMenu(items []list.Item, width int, height int, keyMap config.KeyMappings[key.Binding], options ...Option) Menu {
	m := Menu{
		Items:         items,
		KeyMap:        keyMap,
		FilterMatches: DefaultFilterMatch,
		styles:        createStyles(""),
	}
	for _, opt := range options {
		opt(&m)
	}

	l := list.New(items, MenuItemDelegate{styles: m.styles}, width, height)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(true)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	l.Styles.NoItems = m.styles.dimmed
	l.Styles.PaginationStyle = m.styles.title.Width(10)
	l.Styles.ActivePaginationDot = m.styles.title
	l.Styles.InactivePaginationDot = m.styles.title
	l.FilterInput.PromptStyle = m.styles.matched
	l.FilterInput.Cursor.Style = m.styles.text

	m.List = l
	m.SetWidth(width)
	m.SetHeight(height)

	return m
}

func (m *Menu) Width() int {
	return m.width
}

func (m *Menu) Height() int {
	return m.height
}

func (m *Menu) SetWidth(w int) {
	maxWidth, minWidth := 80, 40
	m.width = max(min(maxWidth, w), minWidth)
	m.List.SetWidth(m.width - 2)
}

func (m *Menu) SetHeight(h int) {
	maxHeight, minHeight := 30, 10
	m.height = max(min(maxHeight, h-2), minHeight)
	m.List.SetHeight(m.height - 2)
}

func (m *Menu) ShowShortcuts(show bool) {
	m.List.SetDelegate(MenuItemDelegate{ShowShortcuts: show, styles: m.styles})
}

func (m *Menu) Filtered(filter string) tea.Cmd {
	m.Filter = filter
	if m.Filter == "" {
		m.List.SetDelegate(MenuItemDelegate{ShowShortcuts: false, styles: m.styles})
		return m.List.SetItems(m.Items)
	}

	m.List.SetDelegate(MenuItemDelegate{ShowShortcuts: true, styles: m.styles})
	var filtered []list.Item
	for _, i := range m.Items {
		if m.FilterMatches(i, m.Filter) {
			filtered = append(filtered, i)
		}
	}
	m.List.ResetSelected()
	return m.List.SetItems(filtered)
}

func (m *Menu) renderFilterView() string {
	filterStyle := m.styles.text.PaddingLeft(1)
	filterValueStyle := m.styles.matched

	filterView := lipgloss.JoinHorizontal(0, filterStyle.Render("Showing "), filterValueStyle.Render("all"))
	if m.Filter != "" {
		filterView = lipgloss.JoinHorizontal(0, filterStyle.Render("Showing only "), filterValueStyle.Render(m.Filter))
	}
	filterViewWidth := lipgloss.Width(filterView)
	paginationView := m.styles.text.AlignHorizontal(1).PaddingRight(1).Width(m.width - filterViewWidth).Render(fmt.Sprintf("%d/%d", m.List.Paginator.Page+1, m.List.Paginator.TotalPages))
	content := lipgloss.JoinHorizontal(0, filterView, paginationView)
	return m.styles.text.Width(m.width).Render(content)
}

func (m *Menu) renderHelpView(helpKeys []key.Binding) string {
	if m.List.SettingFilter() {
		return ""
	}

	bindings := make([]string, 0, len(helpKeys)+1)
	for _, k := range helpKeys {
		if renderedKey := m.renderKey(k); renderedKey != "" {
			bindings = append(bindings, renderedKey)
		}
	}

	if m.List.IsFiltered() {
		bindings = append(bindings, m.renderKey(m.KeyMap.Cancel))
	} else {
		bindings = append(bindings, m.renderKey(m.List.KeyMap.Filter))
	}

	return m.styles.text.PaddingLeft(1).Width(m.width).Render(lipgloss.JoinHorizontal(0, bindings...))
}

func (m *Menu) renderKey(k key.Binding) string {
	if !k.Enabled() {
		return ""
	}
	return lipgloss.JoinHorizontal(0, m.styles.shortcut.Render(k.Help().Key, ""), m.styles.dimmed.Render(k.Help().Desc, ""))
}

func (m *Menu) renderTitle() []string {
	titleView := []string{m.styles.text.Width(m.width).Render(m.styles.title.Render(m.Title))}
	if m.Subtitle != "" {
		titleView = append(titleView, m.styles.text.Width(m.width).Render(m.styles.subtitle.Render(m.Subtitle)))
	}
	return titleView
}

func (m *Menu) View() string {
	views := m.renderTitle()
	views = append(views, "", m.renderFilterView())
	views = append(views, m.List.View())
	content := lipgloss.JoinVertical(0, views...)
	content = lipgloss.Place(m.width, m.height, 0, 0, content)
	content = m.styles.text.Width(m.width).Height(m.height).Render(content)
	return m.styles.border.Render(content)
}
