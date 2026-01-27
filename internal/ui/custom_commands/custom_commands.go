package customcommands

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type item struct {
	name        string
	desc        string
	command     tea.Cmd
	key         key.Binding
	keySequence []key.Binding
}

func (i item) ShortCut() string {
	if len(i.keySequence) > 0 {
		var keys []string
		for _, k := range i.keySequence {
			keys = append(keys, k.Keys()...)
		}
		return strings.Join(keys, " → ")
	}
	k := strings.Join(i.key.Keys(), "/")
	return k
}

func (i item) FilterValue() string {
	return i.name
}

func (i item) Title() string {
	return i.name
}

func (i item) Description() string {
	return i.desc
}

type itemClickMsg struct {
	Index int
}

type itemScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (m itemScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	m.Delta = delta
	m.Horizontal = horizontal
	return m
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

type filterState int

const (
	filterOff filterState = iota
	filterEditing
	filterApplied
)

var _ common.ImmediateModel = (*Model)(nil)

// SortedCustomCommands returns commands ordered by name for deterministic iteration.
func SortedCustomCommands(ctx *context.MainContext) []context.CustomCommand {
	names := make([]string, 0, len(ctx.CustomCommands))
	for name := range ctx.CustomCommands {
		names = append(names, name)
	}
	sort.Strings(names)

	commands := make([]context.CustomCommand, 0, len(names))
	for _, name := range names {
		commands = append(commands, ctx.CustomCommands[name])
	}
	return commands
}

type Model struct {
	context             *context.MainContext
	keymap              config.KeyMappings[key.Binding]
	items               []item
	filteredItems       []item
	cursor              int
	listRenderer        *render.ListRenderer
	filterInput         textinput.Model
	filterState         filterState
	filterText          string
	categoryFilter      string
	showShortcutsBase   bool
	showShortcuts       bool
	ensureCursorVisible bool
	styles              styles
	filterKey           key.Binding
	cancelFilterKey     key.Binding
	acceptFilterKey     key.Binding
	help                help.Model
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keymap.Cancel,
		m.keymap.Apply,
		m.filterKey,
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case itemClickMsg:
		items := m.visibleItems()
		if msg.Index >= 0 && msg.Index < len(items) {
			m.cursor = msg.Index
			m.ensureCursorVisible = true
		}
	case itemScrollMsg:
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
			switch {
			case key.Matches(msg, m.cancelFilterKey):
				m.resetFilter()
				return nil
			case key.Matches(msg, m.acceptFilterKey):
				m.filterText = strings.TrimSpace(m.filterInput.Value())
				if m.filterText == "" {
					m.filterState = filterOff
					m.filterInput.SetValue("")
					m.filterInput.Blur()
				} else {
					m.filterState = filterApplied
					m.filterInput.Blur()
				}
				m.applyFilters(true)
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
		case key.Matches(msg, m.filterKey):
			m.filterState = filterEditing
			m.filterInput.Focus()
			m.filterInput.CursorEnd()
			return textinput.Blink
		case key.Matches(msg, m.keymap.Apply):
			if item, ok := m.selectedItem(); ok {
				return tea.Batch(item.command, common.Close)
			}
		case key.Matches(msg, m.keymap.Cancel):
			if m.hasActiveFilter() {
				m.resetFilter()
				return nil
			}
			return common.Close
		case key.Matches(msg, m.keymap.Up):
			m.moveCursor(-1)
		case key.Matches(msg, m.keymap.Down):
			m.moveCursor(1)
		case key.Matches(msg, m.keymap.ScrollUp):
			m.ensureCursorVisible = false
			m.listRenderer.StartLine -= m.itemHeight()
		case key.Matches(msg, m.keymap.ScrollDown):
			m.ensureCursorVisible = false
			m.listRenderer.StartLine += m.itemHeight()
		default:
			for _, listItem := range m.visibleItems() {
				if key.Matches(msg, listItem.key) {
					return tea.Batch(listItem.command, common.Close)
				}
			}
		}
	}
	return nil
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	pw, ph := box.R.Dx(), box.R.Dy()
	contentWidth := max(min(pw, 80)-4, 0)
	contentHeight := max(min(ph, 40)-4, 0)
	menuWidth := max(contentWidth+2, 0)
	menuHeight := max(contentHeight+2, 0)
	frame := box.Center(menuWidth, menuHeight)
	if frame.R.Dx() <= 0 || frame.R.Dy() <= 0 {
		return
	}

	window := dl.Window(frame.R, 10)
	contentBox := frame.Inset(1)
	if contentBox.R.Dx() <= 0 || contentBox.R.Dy() <= 0 {
		return
	}

	borderBase := lipgloss.NewStyle().Width(contentBox.R.Dx()).Height(contentBox.R.Dy()).Render("")
	window.AddDraw(frame.R, m.styles.border.Render(borderBase), render.ZMenuBorder)

	titleBox, contentBox := contentBox.CutTop(1)
	window.AddDraw(titleBox.R, m.styles.title.Render("Custom Commands"), render.ZMenuContent)

	_, contentBox = contentBox.CutTop(1)
	filterBox, contentBox := contentBox.CutTop(1)
	if m.filterState == filterEditing {
		m.filterInput.Width = max(contentBox.R.Dx()-2, 0)
		window.AddDraw(filterBox.R, m.filterInput.View(), render.ZMenuContent)
	} else {
		m.renderFilterView(window, filterBox)
	}

	_, listBox := contentBox.CutTop(1)
	m.renderList(window, listBox)
}

func NewModel(ctx *context.MainContext) *Model {
	var items []item

	for name, command := range ctx.CustomCommands {
		if command.IsApplicableTo(ctx.SelectedItem) {
			cmd := command.Prepare(ctx)
			desc := command.Description(ctx)
			if lc, ok := command.(context.LabeledCommand); ok {
				desc = lc.Label()
			}
			items = append(items, item{name: name, desc: desc, command: cmd, key: command.Binding(), keySequence: command.Sequence()})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].name < items[j].name
	})

	keyMap := config.Current.GetKeyMap()
	m := &Model{
		context:           ctx,
		keymap:            keyMap,
		items:             items,
		filteredItems:     items,
		listRenderer:      render.NewListRenderer(itemScrollMsg{}),
		styles:            createStyles("custom_commands"),
		showShortcutsBase: true,
		showShortcuts:     true,
		filterKey:         key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		cancelFilterKey:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		acceptFilterKey:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply filter")),
		help:              help.New(),
	}
	m.filterInput = textinput.New()
	m.filterInput.Prompt = "Filter: "
	m.filterInput.PromptStyle = m.styles.matched.PaddingLeft(1)
	m.filterInput.TextStyle = m.styles.text
	m.filterInput.Cursor.Style = m.styles.text
	m.applyFilters(true)
	return m
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

func (m *Model) visibleItems() []item {
	return m.filteredItems
}

func (m *Model) selectedItem() (item, bool) {
	items := m.visibleItems()
	if m.cursor < 0 || m.cursor >= len(items) {
		return item{}, false
	}
	return items[m.cursor], true
}

func (m *Model) itemHeight() int {
	return 3
}

func (m *Model) moveCursor(delta int) {
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

func (m *Model) hasActiveFilter() bool {
	return m.categoryFilter != "" || m.currentFilterText() != ""
}

func (m *Model) currentFilterText() string {
	if m.filterState == filterEditing {
		return strings.TrimSpace(m.filterInput.Value())
	}
	return strings.TrimSpace(m.filterText)
}

func (m *Model) resetFilter() {
	m.filterInput.SetValue("")
	m.filterText = ""
	m.filterState = filterOff
	m.filterInput.Blur()
	m.applyFilters(true)
}

func (m *Model) applyFilters(resetCursor bool) {
	items := m.items
	if m.categoryFilter != "" {
		filtered := make([]item, 0, len(items))
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.FilterValue()), strings.ToLower(m.categoryFilter)) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	filterText := m.currentFilterText()
	if filterText != "" {
		filtered := make([]item, 0, len(items))
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.FilterValue()), strings.ToLower(filterText)) {
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
	m.showShortcuts = m.showShortcutsBase || m.categoryFilter != "" || filterText != ""
}

func (m *Model) renderFilterView(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}
	width := box.R.Dx()
	filterStyle := m.styles.text.PaddingLeft(1)
	filterValueStyle := m.styles.matched

	filterView := lipgloss.JoinHorizontal(0, filterStyle.Render("Showing "), filterValueStyle.Render("all"))
	if m.categoryFilter != "" {
		filterView = lipgloss.JoinHorizontal(0, filterStyle.Render("Showing only "), filterValueStyle.Render(m.categoryFilter))
	}
	dl.AddDraw(box.R, m.styles.text.Width(width).Render(filterView), render.ZMenuContent)
}

func (m *Model) renderList(dl *render.DisplayContext, listBox layout.Box) {
	if listBox.R.Dx() <= 0 || listBox.R.Dy() <= 0 {
		return
	}

	listWidth := max(listBox.R.Dx()-2, 0)
	items := m.visibleItems()
	itemCount := len(items)
	if itemCount == 0 {
		return
	}

	itemHeight := m.itemHeight()
	m.listRenderer.StartLine = render.ClampStartLine(m.listRenderer.StartLine, listBox.R.Dy(), itemCount, itemHeight)
	m.listRenderer.Render(
		dl,
		listBox,
		itemCount,
		m.cursor,
		m.ensureCursorVisible,
		func(_ int) int { return itemHeight },
		func(dl *render.DisplayContext, index int, rect cellbuf.Rectangle) {
			if index < 0 || index >= itemCount {
				return
			}
			renderItem(dl, rect, listWidth, m.styles, m.showShortcuts, m.cursor, index, items[index])
		},
		func(index int) tea.Msg { return itemClickMsg{Index: index} },
	)
	m.listRenderer.RegisterScroll(dl, listBox)
	m.ensureCursorVisible = false
}

func renderItem(dl *render.DisplayContext, rect cellbuf.Rectangle, width int, styles styles, showShortcuts bool, cursor int, index int, item item) {
	var (
		title    string
		desc     string
		shortcut string
	)
	title = item.Title()
	desc = item.Description()
	shortcut = item.ShortCut()
	if width <= 0 {
		return
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

	titleStyle := styles.text
	descStyle := styles.dimmed
	shortcutStyle := styles.shortcut

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

	content := lipgloss.JoinVertical(lipgloss.Left, titleLine, descLine)
	if content == "" {
		return
	}
	dl.AddDraw(rect, content, render.ZMenuContent)
}
