package bookmarks

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"slices"
	"strings"
)

type updateItemsMsg struct {
	items []list.Item
}

type Model struct {
	context context.AppContext
	current *jj.Commit
	filter  string
	list    list.Model
	items   []list.Item
	keymap  config.KeyMappings[key.Binding]
	width   int
	height  int
}

func (m *Model) Width() int {
	return m.width
}

func (m *Model) Height() int {
	return m.height
}

func (m *Model) SetWidth(w int) {
	maxWidth, minWidth := 80, 40
	m.width = max(min(maxWidth, w-4), minWidth)
	m.list.SetWidth(m.width - 8)
}

func (m *Model) SetHeight(h int) {
	maxHeight, minHeight := 30, 10
	m.height = max(min(maxHeight, h-4), minHeight)
	m.list.SetHeight(m.height - 6)
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
	// used to show bookmarks of the selected revision at the top
	weight int
	args   []string
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

func (m *Model) filtered(filter string) (tea.Model, tea.Cmd) {
	m.filter = filter
	if m.filter == "" {
		return m, m.list.SetItems(m.items)
	}
	var filtered []list.Item
	for _, i := range m.items {
		if strings.HasPrefix(i.FilterValue(), m.filter) {
			filtered = append(filtered, i)
		}
	}
	m.list.ResetSelected()
	return m, m.list.SetItems(filtered)
}

func (m *Model) loadMovables() tea.Msg {
	output, _ := m.context.RunCommandImmediate(jj.BookmarkListMovable(m.current.ChangeId))
	var bookmarkItems []list.Item
	bookmarks := jj.ParseBookmarkListOutput(string(output))
	for _, b := range bookmarks {
		if b.Remote || (!b.Conflict && slices.Contains(m.current.Bookmarks, b.Name)) {
			continue
		}

		name := fmt.Sprintf("move '%s' to %s", b.Name, m.current.ChangeId)
		if b.Conflict {
			name = fmt.Sprintf("move conflicted '%s' to %s", b.Name, m.current.ChangeId)
		}
		var extraFlags []string
		if b.Backwards {
			name = fmt.Sprintf("move '%s' backwards to %s", b.Name, m.current.ChangeId)
			extraFlags = append(extraFlags, "--allow-backwards")
		}

		bookmarkItems = append(bookmarkItems, item{
			name:     name,
			priority: moveCommand,
			args:     jj.BookmarkMove(m.current.ChangeId, b.Name, extraFlags...),
		})
	}
	return updateItemsMsg{items: bookmarkItems}
}

func (m *Model) loadAll() tea.Msg {
	if output, err := m.context.RunCommandImmediate(jj.BookmarkListAll()); err != nil {
		return nil
	} else {
		bookmarks := jj.ParseBookmarkListOutput(string(output))

		items := make([]list.Item, 0)
		for _, b := range bookmarks {
			weight := 0
			if slices.Contains(m.current.Bookmarks, b.Name) {
				weight = 1
			}
			if !b.Remote {
				items = append(items, item{
					name:     fmt.Sprintf("delete '%s'", b.Name),
					priority: deleteCommand,
					weight:   weight,
					args:     jj.BookmarkDelete(b.Name),
				})
			}

			items = append(items, item{
				name:     fmt.Sprintf("forget '%s'", b.Name),
				priority: forgetCommand,
				weight:   weight,
				args:     jj.BookmarkForget(b.Name),
			})

			if !b.Tracked && b.Remote {
				items = append(items, item{
					name:     fmt.Sprintf("track '%s'", b.Name),
					priority: trackCommand,
					weight:   weight,
					args:     jj.BookmarkTrack(b.Name),
				})
			}

			if b.Tracked {
				items = append(items, item{
					name:     fmt.Sprintf("untrack '%s'", b.Name),
					priority: untrackCommand,
					weight:   weight,
					args:     jj.BookmarkUntrack(b.Name),
				})
			}
		}
		return updateItemsMsg{items: items}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.SettingFilter() {
			break
		}
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			if m.filter != "" {
				return m.filtered("")
			}
			return m, common.Close
		case key.Matches(msg, m.keymap.Apply):
			if m.list.SelectedItem() == nil {
				break
			}
			action := m.list.SelectedItem().(item)
			return m, m.context.RunCommand(action.args, common.Refresh, common.Close)
		case key.Matches(msg, m.keymap.Bookmark.Move):
			return m.filtered("move")
		case key.Matches(msg, m.keymap.Bookmark.Delete):
			return m.filtered("delete")
		case key.Matches(msg, m.keymap.Bookmark.Forget):
			return m.filtered("forget")
		case key.Matches(msg, m.keymap.Bookmark.Track):
			return m.filtered("track")
		case key.Matches(msg, m.keymap.Bookmark.Untrack):
			return m.filtered("untrack")
		}
	case updateItemsMsg:
		m.items = append(m.items, msg.items...)
		slices.SortFunc(m.items, func(a, b list.Item) int {
			ia := a.(item)
			ib := b.(item)
			if ia.priority < ib.priority {
				return -1
			}
			if ia.priority > ib.priority {
				return 1
			}
			if ia.weight > ib.weight {
				return -1
			}
			if ia.weight < ib.weight {
				return 1
			}
			return strings.Compare(ia.name, ib.name)
		})
		return m, m.list.SetItems(m.items)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

var filterStyle = common.DefaultPalette.ChangeId.PaddingLeft(2)
var filterValueStyle = common.DefaultPalette.Normal.Bold(true)

func (m *Model) View() string {
	title := m.list.Styles.Title.Render(m.list.Title)
	filterView := lipgloss.JoinHorizontal(0, filterStyle.Render("Showing "), filterValueStyle.Render("all"))
	if m.filter != "" {
		filterView = lipgloss.JoinHorizontal(0, filterStyle.Render("Showing only "), filterValueStyle.Render(m.filter))
	}
	listView := m.list.View()
	helpView := m.helpView()
	content := lipgloss.JoinVertical(0, title, "", filterView, "", listView, "", helpView)
	content = lipgloss.Place(m.Width(), m.Height(), 0, 0, content)
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Render(content)
}

func renderKey(k key.Binding) string {
	if !k.Enabled() {
		return ""
	}
	return lipgloss.JoinHorizontal(0, common.DefaultPalette.ChangeId.Render(k.Help().Key, ""), common.DefaultPalette.Dimmed.Render(k.Help().Desc, ""))
}

func (m *Model) helpView() string {
	if m.list.SettingFilter() {
		return ""
	}

	return " " + lipgloss.JoinHorizontal(0,
		renderKey(m.keymap.Cancel),
		renderKey(m.keymap.Apply),
		renderKey(m.keymap.Bookmark.Move),
		renderKey(m.keymap.Bookmark.Delete),
		renderKey(m.keymap.Bookmark.Forget),
		renderKey(m.keymap.Bookmark.Track),
		renderKey(m.keymap.Bookmark.Untrack),
	)
}

func NewModel(c context.AppContext, current *jj.Commit, width int, height int) *Model {
	var items []list.Item
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Bookmark operations"
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()

	m := &Model{
		context: c,
		keymap:  c.KeyMap(),
		list:    l,
		current: current,
	}
	m.SetWidth(width)
	m.SetHeight(height)
	return m
}
