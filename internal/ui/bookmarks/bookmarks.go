package bookmarks

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common/menu"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
)

type updateItemsMsg struct {
	items []list.Item
}

var _ view.IViewModel = (*Model)(nil)

type Model struct {
	*view.ViewNode
	context     *context.MainContext
	current     *models.RevisionItem
	menu        menu.Menu
	keymap      config.KeyMappings[key.Binding]
	distanceMap map[string]int
}

func (m *Model) GetId() view.ViewId {
	return "bookmarks"
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	v.Id = "bookmarks"
	maxWidth, minWidth := 80, 40
	m.Width = max(min(maxWidth, m.ViewManager.Width), minWidth)
	m.menu.SetWidth(m.Width)
	maxHeight, minHeight := 30, 10
	m.Height = max(min(maxHeight, m.ViewManager.Height), minHeight)
	m.menu.SetHeight(m.Height)
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
		m.menu.List.KeyMap.Filter,
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
	args     jj.IGetArgs
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
	desc := strings.Join(i.args.GetArgs(), " ")
	return desc
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.loadAll, m.loadMovables)
}

func (m *Model) filtered(filter string) (tea.Model, tea.Cmd) {
	return m, m.menu.Filtered(filter)
}

func (m *Model) loadMovables() tea.Msg {
	output, _ := m.context.RunCommandImmediate(jj.Args(jj.BookmarkListMovableArgs{Revision: *m.current}))
	var bookmarkItems []list.Item
	bookmarks := jj.ParseBookmarkListOutput(string(output))
	for _, b := range bookmarks {
		if !b.Conflict && b.CommitId == m.current.Commit.CommitId {
			continue
		}

		name := fmt.Sprintf("move '%s' to %s", b.Name, m.current.Commit.GetChangeId())
		if b.Conflict {
			name = fmt.Sprintf("move conflicted '%s' to %s", b.Name, m.current.Commit.GetChangeId())
		}
		if b.Backwards {
			name = fmt.Sprintf("move '%s' backwards to %s", b.Name, m.current.Commit.GetChangeId())
		}
		elem := item{
			name:     name,
			priority: moveCommand,
			args:     jj.BookmarkMoveArgs{Revision: *m.current, Bookmark: b.Name, AllowBackwards: b.Backwards},
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
	if output, err := m.context.RunCommandImmediate(jj.Args(jj.BookmarkListAllArgs{})); err != nil {
		return nil
	} else {
		bookmarks := jj.ParseBookmarkListOutput(string(output))

		items := make([]list.Item, 0)
		for _, b := range bookmarks {
			distance := m.distance(b.CommitId)
			if b.IsDeletable() {
				items = append(items, item{
					name:     fmt.Sprintf("delete '%s'", b.Name),
					priority: deleteCommand,
					dist:     distance,
					args:     jj.BookmarkDeleteArgs{Bookmark: b.Name},
				})
			}

			items = append(items, item{
				name:     fmt.Sprintf("forget '%s'", b.Name),
				priority: forgetCommand,
				dist:     distance,
				args:     jj.BookmarkForgetArgs{Bookmark: b.Name},
			})

			for _, remote := range b.Remotes {
				nameWithRemote := fmt.Sprintf("%s@%s", b.Name, remote.Remote)
				if remote.Tracked {
					items = append(items, item{
						name:     fmt.Sprintf("untrack '%s'", nameWithRemote),
						priority: untrackCommand,
						dist:     distance,
						args:     jj.BookmarkUntrackArgs{Bookmark: b.Name, Remote: remote.Remote},
					})
				} else {
					items = append(items, item{
						name:     fmt.Sprintf("track '%s'", nameWithRemote),
						priority: trackCommand,
						dist:     distance,
						args:     jj.BookmarkTrackArgs{Bookmark: b.Name, Remote: remote.Remote},
					})
				}
			}

		}
		return updateItemsMsg{items: items}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.menu.List.SettingFilter() {
			break
		}
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			if m.menu.Filter != "" || m.menu.List.IsFiltered() {
				m.menu.List.ResetFilter()
				return m.filtered("")
			}
			m.ViewManager.UnregisterView(m.GetId())
			return m, nil
		case key.Matches(msg, m.keymap.Apply):
			if m.menu.List.SelectedItem() == nil {
				break
			}
			action := m.menu.List.SelectedItem().(item)
			m.ViewManager.UnregisterView(m.GetId())
			return m, m.context.RunCommand(action.args.GetArgs(), common.Refresh)
		case key.Matches(msg, m.keymap.Bookmark.Move) && m.menu.Filter != "move":
			return m.filtered("move")
		case key.Matches(msg, m.keymap.Bookmark.Delete) && m.menu.Filter != "delete":
			return m.filtered("delete")
		case key.Matches(msg, m.keymap.Bookmark.Forget) && m.menu.Filter != "forget":
			return m.filtered("forget")
		case key.Matches(msg, m.keymap.Bookmark.Track) && m.menu.Filter != "track":
			return m.filtered("track")
		case key.Matches(msg, m.keymap.Bookmark.Untrack) && m.menu.Filter != "untrack":
			return m.filtered("untrack")
		default:
			for _, listItem := range m.menu.List.Items() {
				if item, ok := listItem.(item); ok && m.menu.Filter != "" && item.key == msg.String() {
					m.ViewManager.UnregisterView(m.GetId())
					return m, m.context.RunCommand(jj.Args(item.args), common.Refresh)
				}
			}
		}
	case updateItemsMsg:
		m.menu.Items = append(m.menu.Items, msg.items...)
		slices.SortFunc(m.menu.Items, itemSorter)
		return m, m.menu.List.SetItems(m.menu.Items)
	}
	var cmd tea.Cmd
	m.menu.List, cmd = m.menu.List.Update(msg)
	return m, cmd
}

func itemSorter(a list.Item, b list.Item) int {
	ia := a.(item)
	ib := b.(item)
	if ia.priority != ib.priority {
		return int(ia.priority) - int(ib.priority)
	}
	if ia.dist == ib.dist {
		return strings.Compare(ia.name, ib.name)
	}
	if ia.dist >= 0 && ib.dist >= 0 {
		return ia.dist - ib.dist
	}
	if ia.dist < 0 && ib.dist < 0 {
		return ib.dist - ia.dist
	}
	return ib.dist - ia.dist
}

func (m *Model) View() string {
	return m.menu.View()
}

func (m *Model) distance(commitId string) int {
	if dist, ok := m.distanceMap[commitId]; ok {
		return dist
	}
	return math.MinInt32
}

func NewModel(c *context.MainContext, current *models.RevisionItem, commitIds []string) view.IViewModel {
	var items []list.Item
	keymap := config.Current.GetKeyMap()
	size := view.NewSizeable(80, 25)

	menu := menu.NewMenu(items, size.Width, size.Height, keymap, menu.WithStylePrefix("bookmarks"))
	menu.Title = "Bookmark Operations"
	menu.FilterMatches = func(i list.Item, filter string) bool {
		return strings.HasPrefix(i.FilterValue(), filter)
	}

	m := &Model{
		context:     c,
		keymap:      keymap,
		menu:        menu,
		current:     current,
		distanceMap: calcDistanceMap(current.Commit.CommitId, commitIds),
	}
	return m
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
