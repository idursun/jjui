// Package helppage provides a help page model for jjui
package helppage

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type helpItem struct {
	display  string
	search   string
	isHeader bool
}

type itemGroup struct {
	groupHeader *helpItem
	groupItems  []helpItem
}

type itemList []itemGroup

type itemMenu struct {
	width, height int
	leftList      itemList
	middleList    itemList
	rightList     itemList
}

type Model struct {
	width        int
	height       int
	keyMap       config.KeyMappings[key.Binding]
	context      *context.MainContext
	styles       styles
	defaultMenu  itemMenu
	filteredMenu itemMenu
	searchQuery  textinput.Model
}

type styles struct {
	border   lipgloss.Style
	title    lipgloss.Style
	text     lipgloss.Style
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
}

func (h *Model) Width() int {
	return h.width
}

func (h *Model) Height() int {
	return h.height
}

func (h *Model) SetWidth(w int) {
	h.width = w
}

func (h *Model) SetHeight(height int) {
	h.height = height
}

func (h *Model) ShortHelp() []key.Binding {
	return []key.Binding{h.keyMap.Help, h.keyMap.Cancel}
}

func (h *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{h.ShortHelp()}
}

func (h *Model) Init() tea.Cmd {
	return textinput.Blink
}

func (h *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, h.keyMap.Help), key.Matches(msg, h.keyMap.Cancel):
			return h, common.Close
		}
	}

	h.searchQuery, cmd = h.searchQuery.Update(msg)
	h.filterMenu()
	return h, cmd
}

func (h *Model) View() string {
	// NOTE: add new lines between search bar and help menu
	content := "\n\n" + h.renderMenu()

	return h.styles.border.Render(h.searchQuery.View(), content)
}

func (h *Model) filterMenu() {
	query := strings.ToLower(strings.TrimSpace(h.searchQuery.Value()))

	if query == "" {
		h.filteredMenu = h.defaultMenu
		return
	}

	h.filteredMenu = itemMenu{
		leftList:   filterList(h.defaultMenu.leftList, query),
		middleList: filterList(h.defaultMenu.middleList, query),
		rightList:  filterList(h.defaultMenu.rightList, query),
	}
}

func filterList(list itemList, query string) itemList {
	var filtered itemList

	for _, group := range list {
		// Check if header matches
		headerMatches := false
		if group.groupHeader != nil {
			headerMatches = strings.Contains(group.groupHeader.search, query)
		}

		if headerMatches {
			filtered = append(filtered, group)
			break
		}

		var matchedItems []helpItem
		for _, item := range group.groupItems {
			if strings.Contains(item.search, query) {
				matchedItems = append(matchedItems, item)
			}
		}

		// Only add group if items matched
		if len(matchedItems) > 0 {
			filtered = append(filtered, itemGroup{
				groupHeader: group.groupHeader,
				groupItems:  matchedItems,
			})
		}
	}

	return filtered
}

func New(context *context.MainContext) *Model {
	styles := styles{
		border:   common.DefaultPalette.GetBorder("help border", lipgloss.NormalBorder()).Padding(1),
		title:    common.DefaultPalette.Get("help title").PaddingLeft(1),
		text:     common.DefaultPalette.Get("help text"),
		dimmed:   common.DefaultPalette.Get("help dimmed").PaddingLeft(1),
		shortcut: common.DefaultPalette.Get("help shortcut"),
	}

	filter := textinput.New()
	filter.Prompt = "Search: "
	filter.Placeholder = "Type to filter..."
	filter.PromptStyle = styles.shortcut
	filter.TextStyle = styles.text
	filter.Cursor.Style = styles.text
	filter.CharLimit = 80
	filter.Focus()

	m := &Model{
		context:     context,
		keyMap:      config.Current.GetKeyMap(),
		styles:      styles,
		searchQuery: filter,
	}

	m.setDefaultMenu()
	return m
}
