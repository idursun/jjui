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
	display    string
	searchTerm string
}

type itemGroup = []helpItem

type menuColumn = []itemGroup

type helpMenu struct {
	width, height int
	leftList      menuColumn
	middleList    menuColumn
	rightList     menuColumn
}

type Model struct {
	width        int
	height       int
	keyMap       config.KeyMappings[key.Binding]
	context      *context.MainContext
	styles       styles
	defaultMenu  helpMenu
	filteredMenu helpMenu
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

	h.filteredMenu = helpMenu{
		leftList:   filterList(h.defaultMenu.leftList, query),
		middleList: filterList(h.defaultMenu.middleList, query),
		rightList:  filterList(h.defaultMenu.rightList, query),
	}
}

func filterList(column menuColumn, query string) menuColumn {
	var filtered menuColumn

	for _, group := range column {
		if len(group) == 0 {
			continue
		}
		// Check if header matches
		header := group[0]
		headerMatches := strings.Contains(header.searchTerm, query)
		if headerMatches {
			filtered = append(filtered, group)
			continue
		}

		matchedItems := []helpItem{header}
		for _, item := range group[1:] {
			if strings.Contains(item.searchTerm, query) {
				matchedItems = append(matchedItems, item)
			}
		}

		if len(matchedItems) > 1 {
			filtered = append(filtered, matchedItems)
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
	filter.PromptStyle = styles.shortcut.PaddingLeft(3)
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
