package git

import (
	"fmt"
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

type itemCategory string

const (
	itemCategoryPush  itemCategory = "push"
	itemCategoryFetch itemCategory = "fetch"
)

// SelectRemoteMsg is sent when a remote is clicked
type SelectRemoteMsg struct {
	Index int
}

type item struct {
	category itemCategory
	key      string
	name     string
	desc     string
	command  []string
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
	keymap              config.KeyMappings[key.Binding]
	items               []item
	filteredItems       []item
	cursor              int
	listRenderer        *render.ListRenderer
	filterInput         textinput.Model
	filterState         filterState
	filterText          string
	categoryFilter      string
	showShortcuts       bool
	ensureCursorVisible bool
	revisions           jj.SelectedRevisions
	remoteNames         []string
	selectedRemoteIdx   int
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
		m.keymap.Quit,
		m.keymap.Apply,
		m.keymap.Git.Push,
		m.keymap.Git.Fetch,
		m.filterKey,
		key.NewBinding(
			key.WithKeys("tab/shift+tab"),
			key.WithHelp("tab/shift+tab", "cycle remotes")),
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
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

	// Remotes are rendered via TextBuilder in ViewRect, no need to update subtitle
	m.items = m.createMenuItems()
	m.applyFilters(false)
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
	case SelectRemoteMsg:
		if msg.Index >= 0 && msg.Index < len(m.remoteNames) {
			m.selectedRemoteIdx = msg.Index
			m.items = m.createMenuItems()
			m.applyFilters(false)
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
			return m.handleIntent(intents.GitCycleRemotes{Delta: 1})
		case msg.Type == tea.KeyShiftTab:
			return m.handleIntent(intents.GitCycleRemotes{Delta: -1})
		case key.Matches(msg, m.filterKey):
			m.filterState = filterEditing
			m.filterInput.Focus()
			m.filterInput.CursorEnd()
			return textinput.Blink
		case key.Matches(msg, m.keymap.Quit):
			return tea.Quit
		case key.Matches(msg, m.keymap.Apply):
			return m.handleIntent(intents.Apply{})
		case key.Matches(msg, m.keymap.Cancel):
			return m.handleIntent(intents.Cancel{})
		case key.Matches(msg, m.keymap.Git.Push) && m.categoryFilter != string(itemCategoryPush):
			return m.handleIntent(intents.GitFilter{Kind: intents.GitFilterPush})
		case key.Matches(msg, m.keymap.Git.Fetch) && m.categoryFilter != string(itemCategoryFetch):
			return m.handleIntent(intents.GitFilter{Kind: intents.GitFilterFetch})
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
			if cmd := m.handleIntent(intents.GitApplyShortcut{Key: msg.String()}); cmd != nil {
				return cmd
			}
		}
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
		return m.context.RunCommand(jj.Args(selected.command...), common.Refresh, common.Close)
	case intents.GitFilter:
		filter := string(msg.Kind)
		if filter != "" && m.categoryFilter != filter {
			m.categoryFilter = filter
			m.applyFilters(true)
		}
	case intents.GitCycleRemotes:
		return m.cycleRemotes(msg.Delta)
	case intents.GitApplyShortcut:
		if m.categoryFilter == "" {
			return nil
		}
		for _, listItem := range m.visibleItems() {
			if listItem.key == msg.Key {
				return m.context.RunCommand(jj.Args(listItem.command...), common.Refresh, common.Close)
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

func (m *Model) filtered(filter string) tea.Cmd {
	m.categoryFilter = filter
	m.applyFilters(true)
	return nil
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

func (m *Model) displayRemotes() string {
	var w strings.Builder
	w.WriteString(m.remoteStyles.promptStyle.PaddingRight(1).Render("Remotes:"))
	if len(m.remoteNames) == 0 {
		w.WriteString(m.remoteStyles.noRemoteStyle.Render("NO REMOTE FOUND"))
		return w.String()
	}
	for idx, remoteName := range m.remoteNames {
		if idx == m.selectedRemoteIdx {
			w.WriteString(m.remoteStyles.selectedStyle.Render(remoteName))
		} else {
			w.WriteString(m.remoteStyles.textStyle.Render(remoteName))
		}
		w.WriteString(" ")
	}
	return w.String()
}

func loadBookmarks(c context.CommandRunner, changeId string) []jj.Bookmark {
	bytes, _ := c.RunCommandImmediate(jj.BookmarkList(changeId))
	bookmarks := jj.ParseBookmarkListOutput(string(bytes))
	return bookmarks
}

func loadRemoteNames(c context.CommandRunner) []string {
	bytes, _ := c.RunCommandImmediate(jj.GitRemoteList())
	remotes := jj.ParseRemoteListOutput(string(bytes))
	return remotes
}

func NewModel(c *context.MainContext, revisions jj.SelectedRevisions) *Model {
	remotes := loadRemoteNames(c)
	keymap := config.Current.GetKeyMap()

	remoteStyles := remoteStyles{
		promptStyle:   common.DefaultPalette.Get("title"),
		textStyle:     common.DefaultPalette.Get("dimmed"),
		selectedStyle: common.DefaultPalette.Get("menu selected"),
		noRemoteStyle: common.DefaultPalette.Get("error"),
	}

	m := &Model{
		context:           c,
		keymap:            keymap,
		revisions:         revisions,
		remoteNames:       remotes,
		selectedRemoteIdx: 0,
		menuStyles:        createMenuStyles("git"),
		remoteStyles:      remoteStyles,
		listRenderer:      render.NewListRenderer(itemScrollMsg{}),
		filterKey:         key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		cancelFilterKey:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		acceptFilterKey:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply filter")),
		title:             "Git Operations",
		subtitle:          " ",
	}

	items := m.createMenuItems()
	m.items = items
	m.filteredItems = items
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
	items := m.items
	if m.categoryFilter != "" {
		filtered := make([]item, 0, len(items))
		for _, item := range items {
			if item.category == itemCategory(m.categoryFilter) {
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
	m.showShortcuts = m.categoryFilter != ""
}

func (m *Model) renderFilterView(dl *render.DisplayContext, box layout.Box) {
	if box.R.Dx() <= 0 || box.R.Dy() <= 0 {
		return
	}
	width := box.R.Dx()
	filterStyle := m.menuStyles.text.PaddingLeft(1)
	filterValueStyle := m.menuStyles.matched

	filterView := lipgloss.JoinHorizontal(0, filterStyle.Render("Showing "), filterValueStyle.Render("all"))
	if m.categoryFilter != "" {
		filterView = lipgloss.JoinHorizontal(0, filterStyle.Render("Showing only "), filterValueStyle.Render(m.categoryFilter))
	}
	dl.AddDraw(box.R, m.menuStyles.text.Width(width).Render(filterView), render.ZMenuContent)
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
	m.listRenderer.StartLine = render.ClampStartLine(m.listRenderer.StartLine, listBox.R.Dy(), itemCount*itemHeight)
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

func (m *Model) createMenuItems() []item {
	revisions := m.revisions
	var items []item
	hasRemote := len(m.remoteNames) > 0
	var selectedRemote string
	if hasRemote {
		selectedRemote = m.remoteNames[m.selectedRemoteIdx]
	} else {
		// set selectedRemote to empty string and `git` command fails gracefully
		selectedRemote = ""
	}

	for _, commit := range revisions.Revisions {
		bookmarks := loadBookmarks(m.context, commit.GetChangeId())
		for _, b := range bookmarks {
			if b.Conflict {
				continue
			}
			for _, remote := range b.Remotes {
				items = append(items, item{
					name:     fmt.Sprintf("git push --bookmark %s --remote %s", b.Name, remote.Remote),
					desc:     fmt.Sprintf("Git push bookmark %s to %s", b.Name, remote.Remote),
					command:  jj.GitPush("--bookmark", b.Name, "--remote", remote.Remote),
					category: itemCategoryPush,
				})
			}
		}
	}
	items = append(items,
		item{
			name:     fmt.Sprintf("git push --remote %s", selectedRemote),
			desc:     "Push tracking bookmarks in the current revset",
			command:  jj.GitPush("--remote", selectedRemote),
			category: itemCategoryPush,
			key:      "p",
		},
		item{
			name:     fmt.Sprintf("git push --all --remote %s", selectedRemote),
			desc:     "Push all bookmarks (including new and deleted bookmarks)",
			command:  jj.GitPush("--all", "--remote", selectedRemote),
			category: itemCategoryPush,
			key:      "a",
		},
	)

	hasMultipleRevisions := len(revisions.Revisions) > 1

	if hasMultipleRevisions {
		flags := []string{"--remote", selectedRemote}
		flags = append(flags, revisions.AsPrefixedArgs("--change")...)
		items = append(items,
			item{
				key:      "c",
				category: itemCategoryPush,
				name:     fmt.Sprintf("git push %s", strings.Join(revisions.AsPrefixedArgs("--change"), " ")),
				desc:     fmt.Sprintf("Push selected changes (%s)", strings.Join(revisions.GetIds(), " ")),
				command:  jj.GitPush(flags...),
			})
	}

	for _, commit := range revisions.Revisions {
		item := item{
			category: itemCategoryPush,
			name:     fmt.Sprintf("git push --change %s --remote %s", commit.GetChangeId(), selectedRemote),
			desc:     fmt.Sprintf("Push the current change (%s)", commit.GetChangeId()),
			command:  jj.GitPush("--change", commit.GetChangeId(), "--remote", selectedRemote),
		}

		if !hasMultipleRevisions {
			item.key = "c"
		}
		items = append(items, item)
	}

	items = append(items,
		item{
			name:     fmt.Sprintf("git push --deleted --remote %s", selectedRemote),
			desc:     "Push all deleted bookmarks",
			command:  jj.GitPush("--deleted", "--remote", selectedRemote),
			category: itemCategoryPush,
			key:      "d",
		},
		item{
			name:     fmt.Sprintf("git push --tracked --remote %s", selectedRemote),
			desc:     "Push all tracked bookmarks (including deleted bookmarks)",
			command:  jj.GitPush("--tracked", "--remote", selectedRemote),
			category: itemCategoryPush,
			key:      "t",
		},
		item{
			name:     fmt.Sprintf("git fetch --remote %s", selectedRemote),
			desc:     "Fetch from remote",
			command:  jj.GitFetch("--remote", selectedRemote),
			category: itemCategoryFetch, key: "f",
		},
		item{
			name:     fmt.Sprintf("git fetch --tracked --remote %s", selectedRemote),
			desc:     "Fetch from remote",
			command:  jj.GitFetch("--tracked", "--remote", selectedRemote),
			category: itemCategoryFetch, key: "f",
		},
		item{
			name:     "git fetch --all-remotes",
			desc:     "Fetch from all remotes",
			command:  jj.GitFetch("--all-remotes"),
			category: itemCategoryFetch,
			key:      "a",
		},
	)

	return items
}
