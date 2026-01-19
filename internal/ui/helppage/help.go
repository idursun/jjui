// Package helppage provides a help page model for jjui
package helppage

import (
	"strings"

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

type scrollMsg struct {
	delta int
}

func (s scrollMsg) SetDelta(delta int, _ bool) tea.Msg {
	s.delta = delta
	return s
}

type helpItem struct {
	display    string
	searchTerm string
}

type itemGroup = []helpItem

type menuColumn = []itemGroup

type helpMenu struct {
	list menuColumn
}

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	keyMap       config.KeyMappings[key.Binding]
	context      *context.MainContext
	styles       styles
	defaultMenu  helpMenu
	filteredMenu helpMenu
	searchQuery  textinput.Model
	renderer     *render.ListRenderer
	cursor       int
}

type styles struct {
	border   lipgloss.Style
	title    lipgloss.Style
	text     lipgloss.Style
	shortcut lipgloss.Style
	dimmed   lipgloss.Style
}

func (h *Model) ShortHelp() []key.Binding {
	return []key.Binding{h.keyMap.Help, h.keyMap.Cancel}
}

func (h *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{h.ShortHelp()}
}

func (h *Model) Init() tea.Cmd {
	return nil
}

func (h *Model) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case scrollMsg:
		h.cursor += msg.delta
		if h.cursor < 0 {
			h.cursor = 0
		}
		allItems := h.getAllItems()
		if h.cursor >= len(allItems) {
			h.cursor = len(allItems) - 1
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, h.keyMap.Cancel):
			return common.Close
		}
	}

	h.searchQuery, cmd = h.searchQuery.Update(msg)
	h.filterMenu()
	return cmd
}

func (h *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	topSection := max(box.R.Dy()-20, 0)
	_, box = box.CutTop(topSection)
	box, _ = box.CutBottom(1)

	dl.AddDraw(box.R, h.styles.border.Width(box.R.Dx()-2).Height(box.R.Dy()-2).Render(""), 2)
	box = box.Inset(1)

	queryBox, box := box.CutTop(2)

	dl.AddDraw(queryBox.R, h.searchQuery.View(), 3)
	allItems := h.getAllItems()

	dl.AddInteraction(box.R, h.renderer.ScrollMsg, render.InteractionScroll, 3)
	h.renderer.Render(dl,
		box,
		len(allItems),
		h.cursor,
		true,
		func(index int) int {
			return 1
		}, func(dl *render.DisplayContext, index int, rect cellbuf.Rectangle) {
			item := allItems[index]
			dl.AddDraw(rect, item.display, 3)
		}, func(index int) render.ClickMessage {
			return nil
		})
}

func (h *Model) getAllItems() []helpItem {
	//TODO: cache this?
	l := h.filteredMenu.list
	if len(l) == 0 {
		l = h.defaultMenu.list
	}

	var allItems []helpItem
	for i, category := range l {
		allItems = append(allItems, category...)
		if i < len(l)-1 {
			allItems = append(allItems, helpItem{}) // spacer between categories
		}
	}
	return allItems
}

func (h *Model) filterMenu() {
	query := strings.ToLower(strings.TrimSpace(h.searchQuery.Value()))

	if query == "" {
		h.filteredMenu = h.defaultMenu
		return
	}

	h.filteredMenu = helpMenu{
		list: filterList(h.defaultMenu.list, query),
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
		border:   common.DefaultPalette.GetBorder("help border", lipgloss.NormalBorder()),
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
		renderer:    render.NewListRenderer(scrollMsg{}),
	}

	m.setDefaultMenu()
	return m
}
