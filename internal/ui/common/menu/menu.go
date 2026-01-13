package menu

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type Menu struct {
	*common.ViewNode
	List          list.Model
	Items         []list.Item
	Filter        string
	KeyMap        config.KeyMappings[key.Binding]
	FilterMatches func(item list.Item, filter string) bool
	Title         string
	Subtitle      string
	styles        styles
	listRenderer  *render.ListRenderer
	showShortcuts bool
	itemDelegate  MenuItemDelegate
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

func NewMenu(items []list.Item, keyMap config.KeyMappings[key.Binding], options ...Option) Menu {
	m := Menu{
		ViewNode:      common.NewViewNode(0, 0),
		Items:         items,
		KeyMap:        keyMap,
		FilterMatches: DefaultFilterMatch,
		styles:        createStyles(""),
		listRenderer:  render.NewListRenderer(nil),
	}
	for _, opt := range options {
		opt(&m)
	}

	l := list.New(items, MenuItemDelegate{styles: m.styles}, 0, 0)
	m.itemDelegate = MenuItemDelegate{styles: m.styles}
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
	return m
}

func (m *Menu) ShowShortcuts(show bool) {
	m.showShortcuts = show
	m.itemDelegate = MenuItemDelegate{ShowShortcuts: show, styles: m.styles}
	m.List.SetDelegate(m.itemDelegate)
}

func (m *Menu) Filtered(filter string) tea.Cmd {
	m.Filter = filter
	if m.Filter == "" {
		m.showShortcuts = false
		m.itemDelegate = MenuItemDelegate{ShowShortcuts: false, styles: m.styles}
		m.List.SetDelegate(m.itemDelegate)
		return m.List.SetItems(m.Items)
	}

	m.showShortcuts = true
	m.itemDelegate = MenuItemDelegate{ShowShortcuts: true, styles: m.styles}
	m.List.SetDelegate(m.itemDelegate)
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
	paginationView := m.styles.text.AlignHorizontal(1).PaddingRight(1).Width(m.Width - filterViewWidth).Render(fmt.Sprintf("%d/%d", m.List.Paginator.Page+1, m.List.Paginator.TotalPages))
	content := lipgloss.JoinHorizontal(0, filterView, paginationView)
	return m.styles.text.Width(m.Width).Render(content)
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

	return m.styles.text.PaddingLeft(1).Width(m.Width).Render(lipgloss.JoinHorizontal(0, bindings...))
}

func (m *Menu) renderKey(k key.Binding) string {
	if !k.Enabled() {
		return ""
	}
	return lipgloss.JoinHorizontal(0, m.styles.shortcut.Render(k.Help().Key, ""), m.styles.dimmed.Render(k.Help().Desc, ""))
}

func (m *Menu) renderTitle() []string {
	titleView := []string{m.styles.text.Width(m.Width).Render(m.styles.title.Render(m.Title))}
	if m.Subtitle != "" {
		titleView = append(titleView, m.styles.text.Width(m.Width).Render(m.styles.subtitle.Render(m.Subtitle)))
	}
	return titleView
}

func (m *Menu) View() string {
	views := m.renderTitle()
	// one empty line for padding between title and content
	views = append(views, "")

	// Calculate remaining height for the list
	remainingHeight := m.Height
	for i := range views {
		remainingHeight -= lipgloss.Height(views[i])
	}
	// reserve space for filter view
	remainingHeight -= 1

	// set list dimensions before rendering filter view so pagination calculates correctly
	m.List.SetWidth(m.Width - 2)
	m.List.SetHeight(remainingHeight)

	views = append(views, m.renderFilterView())
	views = append(views, m.List.View())
	content := lipgloss.JoinVertical(0, views...)
	content = lipgloss.Place(m.Width, m.Height, 0, 0, content)
	return m.styles.border.Render(content)
}

func (m *Menu) ViewRect(dl *render.DisplayList, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}

	contentRect := box.R.Inset(1)
	if contentRect.Dx() <= 0 || contentRect.Dy() <= 0 {
		return
	}

	m.SetFrame(contentRect)

	base := lipgloss.NewStyle().Width(m.Width).Height(m.Height).Render("")
	bordered := m.styles.border.Render(base)
	dl.AddDraw(box.R, bordered, 0)

	var headerLines []string
	headerLines = append(headerLines, m.renderTitle()...)
	headerLines = append(headerLines, "")
	headerLines = append(headerLines, m.renderFilterView())

	if m.List.SettingFilter() {
		filterInput := lipgloss.PlaceHorizontal(m.Width, 0, m.List.FilterInput.View())
		headerLines = append(headerLines, filterInput)
	}

	headerHeight := 0
	for _, line := range headerLines {
		h := lipgloss.Height(line)
		if h == 0 {
			h = 1
			line = lipgloss.NewStyle().Width(m.Width).Render("")
		}
		rect := cellbuf.Rect(contentRect.Min.X, contentRect.Min.Y+headerHeight, m.Width, h)
		dl.AddDraw(rect, line, 1)
		headerHeight += h
	}

	listHeight := m.Height - headerHeight
	if listHeight <= 0 {
		return
	}

	listWidth := max(m.Width-2, 0)
	m.List.SetWidth(listWidth)
	m.List.SetHeight(listHeight)

	items := m.List.VisibleItems()
	itemCount := len(items)
	if itemCount == 0 {
		return
	}

	itemHeight := m.itemDelegate.Height() + m.itemDelegate.Spacing()
	start, _ := m.List.Paginator.GetSliceBounds(itemCount)
	m.listRenderer.StartLine = start * itemHeight

	listRect := layout.Box{R: cellbuf.Rect(contentRect.Min.X, contentRect.Min.Y+headerHeight, m.Width, listHeight)}
	m.listRenderer.Render(
		dl,
		listRect,
		itemCount,
		m.List.Index(),
		false,
		func(_ int) int { return itemHeight },
		func(dl *render.DisplayList, index int, rect cellbuf.Rectangle) {
			if index < 0 || index >= itemCount {
				return
			}
			content := renderMenuItem(m.List, m.styles, m.showShortcuts, index, items[index])
			if content == "" {
				return
			}
			dl.AddDraw(rect, content, 1)
		},
		func(_ int) tea.Msg { return nil },
	)
}
