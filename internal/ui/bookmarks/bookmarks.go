package bookmarks

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type updateItemsMsg struct {
	items []item
}

// SelectRemoteMsg is sent when a remote is clicked
type SelectRemoteMsg struct {
	Index int
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

type menuStyles struct {
	title    lipgloss.Style
	subtitle lipgloss.Style
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
	selected lipgloss.Style
	matched  lipgloss.Style
	text     lipgloss.Style
	border   lipgloss.Style
}

type remoteStyles struct {
	promptStyle   lipgloss.Style
	textStyle     lipgloss.Style
	selectedStyle lipgloss.Style
	noRemoteStyle lipgloss.Style
}

type filterState int

const (
	filterOff filterState = iota
	filterEditing
	filterApplied
)

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	context             *context.MainContext
	current             *jj.Commit
	keymap              config.KeyMappings[key.Binding]
	distanceMap         map[string]int
	remoteNames         []string
	selectedRemoteIdx   int
	allItems            []item
	filteredItems       []item
	cursor              int
	listRenderer        *render.ListRenderer
	filterInput         textinput.Model
	filterState         filterState
	filterText          string
	categoryFilter      string
	showShortcuts       bool
	ensureCursorVisible bool
	menuStyles          menuStyles
	remoteStyles        remoteStyles
	filterKey           key.Binding
	cancelFilterKey     key.Binding
	acceptFilterKey     key.Binding
	title               string
	subtitle            string
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keymap.Cancel,
		m.keymap.Apply,
		m.keymap.Bookmark.Move,
		m.keymap.Bookmark.Delete,
		m.keymap.Bookmark.Forget,
		m.keymap.Bookmark.Track,
		m.keymap.Bookmark.Untrack,
		m.filterKey,
		key.NewBinding(
			key.WithKeys("tab/shift+tab"),
			key.WithHelp("tab/shift+tab", "cycle remotes")),
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

type commandType int

// defines the order of actions in the list
const (
	moveCommand commandType = iota
	deleteCommand
	trackCommand
	untrackCommand
	forgetCommand
)

type item struct {
	name     string
	priority commandType
	dist     int
	args     []string
	key      string
}

func (i item) ShortCut() string {
	return i.key
}

func (i item) FilterValue() string {
	return i.name
}

func (i item) Title() string {
	return i.name
}

func (i item) Description() string {
	desc := strings.Join(i.args, " ")
	return desc
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.loadAll, m.loadMovables)
}

func (m *Model) filtered(filter string) tea.Cmd {
	m.categoryFilter = filter
	m.applyFilters(true)
	return nil
}

func (m *Model) cycleRemotes(step int) tea.Cmd {
	if len(m.remoteNames) == 0 {
		return nil
	}

	m.selectedRemoteIdx += step
	if m.selectedRemoteIdx >= len(m.remoteNames) {
		m.selectedRemoteIdx = 0
	} else if m.selectedRemoteIdx < 0 {
		m.selectedRemoteIdx = len(m.remoteNames) - 1
	}

	return m.updateMenuForRemote()
}

func (m *Model) updateMenuForRemote() tea.Cmd {
	m.applyFilters(false)
	return nil
}

func (m *Model) filterItemsByRemote(allItems []item) []item {
	if len(m.remoteNames) == 0 {
		return allItems
	}

	selectedRemote := m.remoteNames[m.selectedRemoteIdx]
	filtered := make([]item, 0)

	// "local" mode shows local bookmark operations (delete, forget, move, track local, untrack all)
	if selectedRemote == "local" {
		for _, bookmarkItem := range allItems {
			// Include delete, forget, move operations
			if bookmarkItem.priority == deleteCommand || bookmarkItem.priority == forgetCommand || bookmarkItem.priority == moveCommand {
				filtered = append(filtered, bookmarkItem)
				continue
			}

			// Include track items on local bookmarks (no @remote suffix)
			if bookmarkItem.priority == trackCommand && !strings.Contains(bookmarkItem.name, "@") {
				filtered = append(filtered, bookmarkItem)
				continue
			}

			// Include untrack items (bookmarks that are tracked on remotes)
			if bookmarkItem.priority == untrackCommand {
				filtered = append(filtered, bookmarkItem)
				continue
			}
		}
		return filtered
	}

	// Remote mode shows track/untrack items for the selected remote only
	for _, bookmarkItem := range allItems {
		// Only include track/untrack items for the selected remote
		if bookmarkItem.priority == trackCommand || bookmarkItem.priority == untrackCommand {
			if strings.Contains(bookmarkItem.name, "@"+selectedRemote) {
				filtered = append(filtered, bookmarkItem)
			}
		}
	}

	return filtered
}

func (m *Model) loadMovables() tea.Msg {
	output, _ := m.context.RunCommandImmediate(jj.BookmarkListMovable(m.current.GetChangeId()))
	var bookmarkItems []item
	bookmarks := jj.ParseBookmarkListOutput(string(output))
	for _, b := range bookmarks {
		if !b.Conflict && b.CommitId == m.current.CommitId {
			continue
		}

		name := fmt.Sprintf("move '%s' to %s", b.Name, m.current.GetChangeId())
		if b.Conflict {
			name = fmt.Sprintf("move conflicted '%s' to %s", b.Name, m.current.GetChangeId())
		}
		var extraFlags []string
		if b.Backwards {
			name = fmt.Sprintf("move '%s' backwards to %s", b.Name, m.current.GetChangeId())
			extraFlags = append(extraFlags, "--allow-backwards")
		}
		elem := item{
			name:     name,
			priority: moveCommand,
			args:     jj.BookmarkMove(m.current.GetChangeId(), b.Name, extraFlags...),
			dist:     m.distance(b.CommitId),
		}
		if b.Name == "main" || b.Name == "master" {
			elem.key = "m"
		}
		bookmarkItems = append(bookmarkItems, elem)
	}
	return updateItemsMsg{items: bookmarkItems}
}

func (m *Model) loadAll() tea.Msg {
	if output, err := m.context.RunCommandImmediate(jj.BookmarkListAll()); err != nil {
		return nil
	} else {
		bookmarks := jj.ParseBookmarkListOutput(string(output))

		items := make([]item, 0)
		for _, b := range bookmarks {
			distance := m.distance(b.CommitId)
			if b.IsDeletable() {
				items = append(items, item{
					name:     fmt.Sprintf("delete '%s'", b.Name),
					priority: deleteCommand,
					dist:     distance,
					args:     jj.BookmarkDelete(b.Name),
				})
			}

			items = append(items, item{
				name:     fmt.Sprintf("forget '%s'", b.Name),
				priority: forgetCommand,
				dist:     distance,
				args:     jj.BookmarkForget(b.Name),
			})

			// Track local bookmarks as they have no remotes
			if b.IsTrackable() {
				items = append(items, item{
					name:     fmt.Sprintf("track '%s'", b.Name),
					priority: trackCommand,
					dist:     distance,
					args:     jj.BookmarkTrack(b.Name, ""),
				})
			}

			for _, remote := range b.Remotes {
				nameWithRemote := fmt.Sprintf("%s@%s", b.Name, remote.Remote)
				if remote.Tracked {
					items = append(items, item{
						name:     fmt.Sprintf("untrack '%s'", nameWithRemote),
						priority: untrackCommand,
						dist:     distance,
						args:     jj.BookmarkUntrack(b.Name, remote.Remote),
					})
				} else {
					items = append(items, item{
						name:     fmt.Sprintf("track '%s'", nameWithRemote),
						priority: trackCommand,
						dist:     distance,
						args:     jj.BookmarkTrack(b.Name, remote.Remote),
					})
				}
			}
		}
		return updateItemsMsg{items: items}
	}
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
	case SelectRemoteMsg:
		if msg.Index >= 0 && msg.Index < len(m.remoteNames) {
			m.selectedRemoteIdx = msg.Index
			return m.updateMenuForRemote()
		}
		return nil
	case intents.Intent:
		return m.handleIntent(msg)
	case tea.KeyMsg:
		if m.filterState == filterEditing {
			switch {
			case key.Matches(msg, m.cancelFilterKey):
				m.resetTextFilter()
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
		case msg.Type == tea.KeyTab:
			return m.handleIntent(intents.BookmarksCycleRemotes{Delta: 1})
		case msg.Type == tea.KeyShiftTab:
			return m.handleIntent(intents.BookmarksCycleRemotes{Delta: -1})
		case key.Matches(msg, m.filterKey):
			m.filterState = filterEditing
			m.filterInput.Focus()
			m.filterInput.CursorEnd()
			return textinput.Blink
		case key.Matches(msg, m.keymap.Cancel):
			return m.handleIntent(intents.Cancel{})
		case key.Matches(msg, m.keymap.Apply):
			return m.handleIntent(intents.Apply{})
		case key.Matches(msg, m.keymap.Bookmark.Move) && m.categoryFilter != "move":
			return m.handleIntent(intents.BookmarksFilter{Kind: intents.BookmarksFilterMove})
		case key.Matches(msg, m.keymap.Bookmark.Delete) && m.categoryFilter != "delete":
			return m.handleIntent(intents.BookmarksFilter{Kind: intents.BookmarksFilterDelete})
		case key.Matches(msg, m.keymap.Bookmark.Forget) && m.categoryFilter != "forget":
			return m.handleIntent(intents.BookmarksFilter{Kind: intents.BookmarksFilterForget})
		case key.Matches(msg, m.keymap.Bookmark.Track) && m.categoryFilter != "track":
			return m.handleIntent(intents.BookmarksFilter{Kind: intents.BookmarksFilterTrack})
		case key.Matches(msg, m.keymap.Bookmark.Untrack) && m.categoryFilter != "untrack":
			return m.handleIntent(intents.BookmarksFilter{Kind: intents.BookmarksFilterUntrack})
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
			if cmd := m.handleIntent(intents.BookmarksApplyShortcut{Key: msg.String()}); cmd != nil {
				return cmd
			}
		}
	case updateItemsMsg:
		m.allItems = append(m.allItems, msg.items...)
		slices.SortFunc(m.allItems, itemSorter)
		return m.updateMenuForRemote()
	}
	return nil
}

func (m *Model) handleIntent(intent intents.Intent) tea.Cmd {
	switch msg := intent.(type) {
	case intents.Apply:
		selected, ok := m.selectedItem()
		if !ok {
			return nil
		}
		return m.context.RunCommand(selected.args, common.Refresh, common.Close)
	case intents.BookmarksFilter:
		filter := string(msg.Kind)
		if filter != "" && m.categoryFilter != filter {
			return m.filtered(filter)
		}
	case intents.BookmarksCycleRemotes:
		return m.cycleRemotes(msg.Delta)
	case intents.BookmarksApplyShortcut:
		if m.categoryFilter == "" {
			return nil
		}
		for _, listItem := range m.visibleItems() {
			if listItem.key == msg.Key {
				return m.context.RunCommand(jj.Args(listItem.args...), common.Refresh, common.Close)
			}
		}
		return nil
	case intents.Cancel:
		if m.hasActiveFilter() {
			m.resetAllFilters()
			return nil
		}
		return common.Close
	}
	return nil
}

func itemSorter(a item, b item) int {
	if a.priority != b.priority {
		return int(a.priority) - int(b.priority)
	}
	if a.dist == b.dist {
		return strings.Compare(a.name, b.name)
	}
	if a.dist >= 0 && b.dist >= 0 {
		return a.dist - b.dist
	}
	if a.dist < 0 && b.dist < 0 {
		return b.dist - a.dist
	}
	return b.dist - a.dist
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	pw, ph := box.R.Dx(), box.R.Dy()
	contentWidth := max(min(pw, 80)-4, 0)
	contentHeight := max(min(ph, 40)-4, 0)
	menuWidth := max(contentWidth+2, 0)
	menuHeight := max(contentHeight+2, 0)
	frame := box.Center(menuWidth, menuHeight)
	if len(m.visibleItems()) == 0 {
		dl.AddFill(frame.R.Inset(1), ' ', lipgloss.NewStyle(), render.ZMenuContent)
	}
	if frame.R.Dx() <= 0 || frame.R.Dy() <= 0 {
		return
	}

	window := dl.Window(frame.R, 10)
	contentBox := frame.Inset(1)
	if contentBox.R.Dx() <= 0 || contentBox.R.Dy() <= 0 {
		return
	}

	borderBase := lipgloss.NewStyle().Width(contentBox.R.Dx()).Height(contentBox.R.Dy()).Render("")
	window.AddDraw(frame.R, m.menuStyles.border.Render(borderBase), render.ZMenuBorder)

	titleBox, contentBox := contentBox.CutTop(1)
	window.AddDraw(titleBox.R, m.menuStyles.title.Render(m.title), render.ZMenuContent)

	_, contentBox = contentBox.CutTop(1)
	remoteBox, contentBox := contentBox.CutTop(1)
	m.renderRemotes(window, remoteBox)

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

func (m *Model) renderRemotes(dl *render.DisplayContext, lineBox layout.Box) {
	// Create a window for remotes with higher z-index than menu
	// so that clicks are routed to this window instead of the menu
	if lineBox.R.Dx() <= 0 || lineBox.R.Dy() <= 0 {
		return
	}
	windowedDl := dl.Window(lineBox.R, render.ZMenuContent)

	// Render above menu content
	tb := windowedDl.Text(lineBox.R.Min.X, lineBox.R.Min.Y, render.ZMenuContent+1).
		Space(1).
		Styled("Remotes: ", m.remoteStyles.promptStyle)

	if len(m.remoteNames) == 0 {
		tb.Styled("NO REMOTE FOUND", m.remoteStyles.noRemoteStyle).Done()
		return
	}

	for idx, remoteName := range m.remoteNames {
		style := m.remoteStyles.textStyle
		if idx == m.selectedRemoteIdx {
			style = m.remoteStyles.selectedStyle
		}
		tb.Clickable(remoteName, style, SelectRemoteMsg{Index: idx}).Space(1)
	}

	tb.Done()
}

func (m *Model) distance(commitId string) int {
	if dist, ok := m.distanceMap[commitId]; ok {
		return dist
	}
	return math.MinInt32
}

func loadRemoteNames(c context.CommandRunner) []string {
	bytes, _ := c.RunCommandImmediate(jj.GitRemoteList())
	remotes := jj.ParseRemoteListOutput(string(bytes))
	return remotes
}

func NewModel(c *context.MainContext, current *jj.Commit, commitIds []string) *Model {
	keymap := config.Current.GetKeyMap()
	remotes := loadRemoteNames(c)
	// Add "local" as the first option to view local bookmark operations
	remotes = append([]string{"local"}, remotes...)

	remoteStyles := remoteStyles{
		promptStyle:   common.DefaultPalette.Get("title"),
		textStyle:     common.DefaultPalette.Get("dimmed"),
		selectedStyle: common.DefaultPalette.Get("menu selected"),
		noRemoteStyle: common.DefaultPalette.Get("error"),
	}

	m := &Model{
		context:           c,
		keymap:            keymap,
		current:           current,
		distanceMap:       calcDistanceMap(current.CommitId, commitIds),
		remoteNames:       remotes,
		selectedRemoteIdx: 0,
		remoteStyles:      remoteStyles,
		menuStyles:        createMenuStyles("bookmarks"),
		listRenderer:      render.NewListRenderer(itemScrollMsg{}),
		filterKey:         key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		cancelFilterKey:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		acceptFilterKey:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply filter")),
		title:             "Bookmark Operations",
		subtitle:          " ",
		allItems:          make([]item, 0),
	}

	m.filterInput = textinput.New()
	m.filterInput.Prompt = "Filter: "
	m.filterInput.PromptStyle = m.menuStyles.matched.PaddingLeft(1)
	m.filterInput.TextStyle = m.menuStyles.text
	m.filterInput.Cursor.Style = m.menuStyles.text
	m.applyFilters(true)

	return m
}

func createMenuStyles(prefix string) menuStyles {
	if prefix != "" {
		prefix += " "
	}
	return menuStyles{
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

func (m *Model) resetTextFilter() {
	m.filterInput.SetValue("")
	m.filterText = ""
	m.filterState = filterOff
	m.filterInput.Blur()
	m.applyFilters(true)
}

func (m *Model) resetAllFilters() {
	m.categoryFilter = ""
	m.resetTextFilter()
}

func (m *Model) applyFilters(resetCursor bool) {
	items := m.filterItemsByRemote(m.allItems)

	if m.categoryFilter != "" {
		filtered := make([]item, 0, len(items))
		for _, item := range items {
			if m.categoryFilterMatch(item, m.categoryFilter) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	filterText := m.currentFilterText()
	if filterText != "" {
		filtered := make([]item, 0, len(items))
		for _, item := range items {
			if m.textFilterMatch(item, filterText) {
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
	m.showShortcuts = m.categoryFilter != ""
}

func (m *Model) categoryFilterMatch(item item, filter string) bool {
	if !strings.HasPrefix(item.FilterValue(), filter) {
		return false
	}
	if strings.HasPrefix(filter, "track") || strings.HasPrefix(filter, "untrack") {
		return m.remoteFilterMatch(item, filter)
	}
	return true
}

func (m *Model) textFilterMatch(item item, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	lowerFilter := strings.ToLower(filter)
	if !strings.Contains(strings.ToLower(item.FilterValue()), lowerFilter) {
		return false
	}
	if strings.HasPrefix(lowerFilter, "track") || strings.HasPrefix(lowerFilter, "untrack") {
		return m.remoteFilterMatch(item, lowerFilter)
	}
	return true
}

func (m *Model) remoteFilterMatch(item item, filter string) bool {
	if len(m.remoteNames) == 0 || m.selectedRemoteIdx >= len(m.remoteNames) {
		return true
	}
	selectedRemote := m.remoteNames[m.selectedRemoteIdx]
	if selectedRemote == "local" {
		if strings.HasPrefix(filter, "untrack") {
			return true
		}
		return !strings.Contains(item.FilterValue(), "@")
	}
	if strings.Contains(item.FilterValue(), "@") {
		return strings.Contains(item.FilterValue(), "@"+selectedRemote)
	}
	return false
}

func (m *Model) renderFilterView(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}
	width := box.R.Dx()
	labelStyle := m.menuStyles.text.PaddingLeft(1).PaddingRight(1)
	valueStyle := m.menuStyles.matched

	action := "all actions"
	if m.categoryFilter != "" {
		action = m.categoryFilter
	}

	remote := "all remotes"
	if len(m.remoteNames) > 0 && m.selectedRemoteIdx < len(m.remoteNames) {
		remote = m.remoteNames[m.selectedRemoteIdx]
	}

	parts := []string{
		labelStyle.Render("Showing:"),
		valueStyle.Render(action),
		labelStyle.Render("for"),
		valueStyle.Render(remote),
		labelStyle.Render("remote"),
	}

	filterText := m.currentFilterText()
	if filterText != "" {
		parts = append(parts,
			labelStyle.Render("containing"),
			valueStyle.Render("\""+filterText+"\""),
		)
	}

	dl.AddDraw(box.R, m.menuStyles.text.Width(width).Render(lipgloss.JoinHorizontal(0, parts...)), render.ZMenuContent)
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
			renderItem(dl, rect, listWidth, m.menuStyles, m.showShortcuts, m.cursor, index, items[index])
		},
		func(index int) tea.Msg { return itemClickMsg{Index: index} },
	)
	m.listRenderer.RegisterScroll(dl, listBox)
	m.ensureCursorVisible = false
}

func renderItem(dl *render.DisplayContext, rect cellbuf.Rectangle, width int, styles menuStyles, showShortcuts bool, cursor int, index int, item item) {
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

func calcDistanceMap(current string, commitIds []string) map[string]int {
	distanceMap := make(map[string]int)
	currentPos := -1
	for i, id := range commitIds {
		if id == current {
			currentPos = i
			break
		}
	}
	for i, id := range commitIds {
		dist := i - currentPos
		distanceMap[id] = dist
	}
	return distanceMap
}
