package menu

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type Menu struct {
	Items               []Item
	filteredItems       []Item
	Filter              string
	KeyMap              config.KeyMappings[key.Binding]
	FilterMatches       func(item Item, filter string) bool
	TextFilterMatches   func(item Item, filter string) bool
	Title               string
	Subtitle            string
	styles              styles
	listRenderer        *render.ListRenderer
	showShortcuts       bool
	showShortcutsBase   bool
	cursor              int
	filterInput         textinput.Model
	filterState         filterState
	ensureCursorVisible bool
	FilterKey           key.Binding
	cancelFilterKey     key.Binding
	acceptFilterKey     key.Binding
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

type FilterMatchFunc func(Item, string) bool

type filterState int

const (
	filterOff filterState = iota
	filterEditing
	filterApplied
)

// Z-index constants for menu overlays.
// Menu overlays render above main content (z=0-1) to ensure visibility.
const (
	ZIndexBorder  = 100 // Z-index for menu border
	ZIndexContent = 101 // Z-index for menu content
)

type MenuClickMsg struct {
	Index int
}

type MenuScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (m MenuScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	m.Delta = delta
	m.Horizontal = horizontal
	return m
}

func DefaultFilterMatch(item Item, filter string) bool {
	return true
}

func DefaultTextFilterMatch(item Item, filter string) bool {
	return strings.Contains(strings.ToLower(item.FilterValue()), strings.ToLower(filter))
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

func NewMenu(items []Item, keyMap config.KeyMappings[key.Binding], options ...Option) Menu {
	m := Menu{
		Items:             items,
		KeyMap:            keyMap,
		FilterMatches:     DefaultFilterMatch,
		TextFilterMatches: DefaultTextFilterMatch,
		styles:            createStyles(""),
		listRenderer:      render.NewListRenderer(MenuScrollMsg{}),
		FilterKey: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		cancelFilterKey: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		acceptFilterKey: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "apply filter"),
		),
	}
	for _, opt := range options {
		opt(&m)
	}

	m.filteredItems = items
	m.filterInput = textinput.New()
	m.filterInput.Prompt = "Filter: "
	m.filterInput.PromptStyle = m.styles.matched
	m.filterInput.TextStyle = m.styles.text
	m.filterInput.Cursor.Style = m.styles.text
	return m
}

func (m *Menu) ShowShortcuts(show bool) {
	m.showShortcutsBase = show
	m.showShortcuts = show || m.Filter != ""
}

func (m *Menu) Filtered(filter string) tea.Cmd {
	m.Filter = filter
	m.showShortcuts = m.showShortcutsBase || m.Filter != ""
	m.applyFilters(true)
	return nil
}

func (m *Menu) SetItems(items []Item) tea.Cmd {
	m.Items = items
	m.applyFilters(false)
	return nil
}

func (m *Menu) SelectedItem() Item {
	items := m.visibleItems()
	if m.cursor < 0 || m.cursor >= len(items) {
		return nil
	}
	return items[m.cursor]
}

func (m *Menu) VisibleItems() []Item {
	return m.visibleItems()
}

func (m *Menu) SettingFilter() bool {
	return m.filterState == filterEditing
}

func (m *Menu) IsFiltered() bool {
	return m.filterState == filterApplied
}

func (m *Menu) ResetFilter() {
	m.filterInput.SetValue("")
	m.filterState = filterOff
	m.filterInput.Blur()
	m.applyFilters(true)
}

func (m *Menu) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case MenuClickMsg:
		items := m.visibleItems()
		if msg.Index >= 0 && msg.Index < len(items) {
			m.cursor = msg.Index
			m.ensureCursorVisible = true
		}
	case MenuScrollMsg:
		if msg.Horizontal {
			return nil
		}
		m.ensureCursorVisible = false
		m.listRenderer.StartLine += msg.Delta
		if m.listRenderer.StartLine < 0 {
			m.listRenderer.StartLine = 0
		}
	case tea.KeyMsg:
		if m.filterState == filterEditing {
			if key.Matches(msg, m.cancelFilterKey) {
				m.ResetFilter()
				return nil
			}
			if key.Matches(msg, m.acceptFilterKey) {
				if m.filterInput.Value() == "" {
					m.ResetFilter()
					return nil
				}
				m.filterState = filterApplied
				m.filterInput.Blur()
				return nil
			}
			updated, cmd := m.filterInput.Update(msg)
			filterChanged := m.filterInput.Value() != updated.Value()
			m.filterInput = updated
			if filterChanged {
				m.applyFilters(false)
			}
			return cmd
		}
		switch {
		case key.Matches(msg, m.FilterKey):
			m.filterState = filterEditing
			m.filterInput.Focus()
			m.filterInput.CursorEnd()
			return nil
		case key.Matches(msg, m.KeyMap.Up):
			m.moveCursor(-1)
		case key.Matches(msg, m.KeyMap.Down):
			m.moveCursor(1)
		case key.Matches(msg, m.KeyMap.ScrollUp):
			m.ensureCursorVisible = false
			m.listRenderer.StartLine -= m.itemHeight()
		case key.Matches(msg, m.KeyMap.ScrollDown):
			m.ensureCursorVisible = false
			m.listRenderer.StartLine += m.itemHeight()
		}
	}
	return nil
}

func (m *Menu) renderFilterView(width int) string {
	filterStyle := m.styles.text.PaddingLeft(1)
	filterValueStyle := m.styles.matched

	filterView := lipgloss.JoinHorizontal(0, filterStyle.Render("Showing "), filterValueStyle.Render("all"))
	if m.Filter != "" {
		filterView = lipgloss.JoinHorizontal(0, filterStyle.Render("Showing only "), filterValueStyle.Render(m.Filter))
	}
	return m.styles.text.Width(width).Render(filterView)
}

func (m *Menu) renderHelpView(width int, helpKeys []key.Binding) string {
	if m.SettingFilter() {
		return ""
	}

	bindings := make([]string, 0, len(helpKeys)+1)
	for _, k := range helpKeys {
		if renderedKey := m.renderKey(k); renderedKey != "" {
			bindings = append(bindings, renderedKey)
		}
	}

	if m.IsFiltered() {
		bindings = append(bindings, m.renderKey(m.KeyMap.Cancel))
	} else {
		bindings = append(bindings, m.renderKey(m.FilterKey))
	}

	return m.styles.text.PaddingLeft(1).Width(width).Render(lipgloss.JoinHorizontal(0, bindings...))
}

func (m *Menu) renderKey(k key.Binding) string {
	if !k.Enabled() {
		return ""
	}
	return lipgloss.JoinHorizontal(0, m.styles.shortcut.Render(k.Help().Key, ""), m.styles.dimmed.Render(k.Help().Desc, ""))
}

func (m *Menu) renderTitle(width int) []string {
	titleView := []string{m.styles.text.Width(width).Render(m.styles.title.Render(m.Title))}
	if m.Subtitle != "" {
		titleView = append(titleView, m.styles.text.Width(width).Render(m.styles.subtitle.Render(m.Subtitle)))
	}
	return titleView
}

func (m *Menu) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}

	dl = dl.Window(box.R, 10)

	contentRect := box.R.Inset(1)
	if contentRect.Dx() <= 0 || contentRect.Dy() <= 0 {
		return
	}

	contentWidth := contentRect.Dx()
	contentHeight := contentRect.Dy()

	base := lipgloss.NewStyle().Width(contentWidth).Height(contentHeight).Render("")
	bordered := m.styles.border.Render(base)
	dl.AddDraw(box.R, bordered, ZIndexBorder)

	var headerLines []string
	headerLines = append(headerLines, m.renderTitle(contentWidth)...)
	headerLines = append(headerLines, "")
	headerLines = append(headerLines, m.renderFilterView(contentWidth))

	if m.SettingFilter() {
		m.filterInput.Width = max(contentWidth-2, 0)
		filterInput := lipgloss.PlaceHorizontal(contentWidth, 0, m.filterInput.View())
		headerLines = append(headerLines, filterInput)
	}

	headerHeight := 0
	for _, line := range headerLines {
		h := lipgloss.Height(line)
		if h == 0 {
			h = 1
			line = lipgloss.NewStyle().Width(contentWidth).Render("")
		}
		rect := cellbuf.Rect(contentRect.Min.X, contentRect.Min.Y+headerHeight, contentWidth, h)
		dl.AddDraw(rect, line, ZIndexContent)
		headerHeight += h
	}

	listHeight := contentHeight - headerHeight
	if listHeight <= 0 {
		return
	}

	listWidth := max(contentWidth-2, 0)
	items := m.visibleItems()
	itemCount := len(items)
	if itemCount == 0 {
		return
	}

	itemHeight := m.itemHeight()
	m.clampScroll(listHeight, itemCount, itemHeight)

	listRect := layout.Box{R: cellbuf.Rect(contentRect.Min.X, contentRect.Min.Y+headerHeight, contentWidth, listHeight)}
	m.listRenderer.Render(
		dl,
		listRect,
		itemCount,
		m.cursor,
		m.ensureCursorVisible,
		func(_ int) int { return itemHeight },
		func(dl *render.DisplayContext, index int, rect cellbuf.Rectangle) {
			if index < 0 || index >= itemCount {
				return
			}
			content := renderMenuItem(listWidth, m.styles, m.showShortcuts, m.cursor, index, items[index])
			if content == "" {
				return
			}
			dl.AddDraw(rect, content, ZIndexContent)
		},
		func(index int) tea.Msg { return MenuClickMsg{Index: index} },
	)
	m.listRenderer.RegisterScroll(dl, listRect)
	m.ensureCursorVisible = false
}

func (m *Menu) visibleItems() []Item {
	return m.filteredItems
}

func (m *Menu) itemHeight() int {
	return 3
}

func (m *Menu) moveCursor(delta int) {
	items := m.visibleItems()
	if len(items) == 0 {
		m.cursor = 0
		return
	}
	next := m.cursor + delta
	if next < 0 {
		next = 0
	} else if next >= len(items) {
		next = len(items) - 1
	}
	if next != m.cursor {
		m.cursor = next
		m.ensureCursorVisible = true
	}
}

func (m *Menu) applyFilters(resetCursor bool) {
	items := m.Items
	if m.Filter != "" {
		filtered := make([]Item, 0, len(items))
		for _, item := range items {
			if m.FilterMatches(item, m.Filter) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	filterText := strings.TrimSpace(m.filterInput.Value())
	if filterText != "" {
		filtered := make([]Item, 0, len(items))
		for _, item := range items {
			if m.TextFilterMatches(item, filterText) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	m.filteredItems = items
	if resetCursor || m.cursor >= len(m.filteredItems) {
		m.cursor = 0
	}
	m.listRenderer.StartLine = 0
}

func (m *Menu) clampScroll(listHeight int, itemCount int, itemHeight int) {
	if m.listRenderer.StartLine < 0 {
		m.listRenderer.StartLine = 0
	}
	totalLines := itemCount * itemHeight
	maxStart := totalLines - listHeight
	if maxStart < 0 {
		maxStart = 0
	}
	if m.listRenderer.StartLine > maxStart {
		m.listRenderer.StartLine = maxStart
	}
}
