package git

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/menu"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
)

type itemCategory string

const (
	itemCategoryPush  itemCategory = "push"
	itemCategoryFetch itemCategory = "fetch"
)

type item struct {
	category itemCategory
	key      string
	name     string
	desc     string
	command  jj.IGetArgs
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

var _ view.IViewModel = (*Model)(nil)

type Model struct {
	*view.ViewNode
	context *context.MainContext
	keymap  config.KeyMappings[key.Binding]
	menu    menu.Menu
}

func (m *Model) GetId() view.ViewId {
	return "git"
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	v.Id = "git"
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
		m.keymap.Git.Push,
		m.keymap.Git.Fetch,
		m.menu.List.KeyMap.Filter,
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.menu.List.SettingFilter() {
			break
		}
		switch {
		case key.Matches(msg, m.keymap.Apply):
			action := m.menu.List.SelectedItem().(item)
			m.ViewManager.UnregisterView(m.GetId())
			return m, m.context.RunCommand(jj.Args(action.command), common.Refresh)
		case key.Matches(msg, m.keymap.Cancel):
			if m.menu.Filter != "" || m.menu.List.IsFiltered() {
				m.menu.List.ResetFilter()
				return m.filtered("")
			}
			m.ViewManager.UnregisterView(m.GetId())
			return m, nil
		case key.Matches(msg, m.keymap.Git.Push) && m.menu.Filter != string(itemCategoryPush):
			return m.filtered(string(itemCategoryPush))
		case key.Matches(msg, m.keymap.Git.Fetch) && m.menu.Filter != string(itemCategoryFetch):
			return m.filtered(string(itemCategoryFetch))
		default:
			for _, listItem := range m.menu.List.Items() {
				if item, ok := listItem.(item); ok && m.menu.Filter != "" && item.key == msg.String() {
					m.ViewManager.UnregisterView(m.GetId())
					return m, m.context.RunCommand(jj.Args(item.command), common.Refresh)
				}
			}
		}
	}
	var cmd tea.Cmd
	m.menu.List, cmd = m.menu.List.Update(msg)
	return m, cmd
}

func (m *Model) filtered(filter string) (tea.Model, tea.Cmd) {
	return m, m.menu.Filtered(filter)
}

func (m *Model) View() string {
	return m.menu.View()
}

func loadBookmarks(c context.CommandRunner, revision *models.RevisionItem) []jj.Bookmark {
	bytes, _ := c.RunCommandImmediate(jj.Args(jj.BookmarkListArgs{Revision: *revision}))
	bookmarks := jj.ParseBookmarkListOutput(string(bytes))
	return bookmarks
}

func NewModel(c *context.MainContext, revision *models.RevisionItem) view.IViewModel {
	var items []list.Item
	if revision != nil {
		bookmarks := loadBookmarks(c, revision)
		for _, b := range bookmarks {
			if b.Conflict {
				continue
			}
			for _, remote := range b.Remotes {
				items = append(items, item{
					name:     fmt.Sprintf("git push --bookmark %s --remote %s", b.Name, remote.Remote),
					desc:     fmt.Sprintf("Git push bookmark %s to remote %s", b.Name, remote.Remote),
					command:  jj.GitPushCommandArgs{Bookmark: b.Name, Remote: remote.Remote},
					category: itemCategoryPush,
				})
			}
			if b.IsPushable() {
				items = append(items, item{
					name:     fmt.Sprintf("git push --bookmark %s --allow-new", b.Name),
					desc:     fmt.Sprintf("Git push new bookmark %s", b.Name),
					command:  jj.GitPushCommandArgs{Bookmark: b.Name, AllowNew: true},
					category: itemCategoryPush,
				})
			}
		}
	}
	items = append(items,
		item{name: "git push", desc: "Push tracking bookmarks in the current revset", command: jj.GitPushCommandArgs{}, category: itemCategoryPush, key: "p"},
		item{name: "git push --all", desc: "Push all bookmarks (including new and deleted bookmarks)", command: jj.GitPushCommandArgs{All: true}, category: itemCategoryPush, key: "a"},
	)
	if revision != nil {
		items = append(items,
			item{
				key:      "c",
				category: itemCategoryPush,
				name:     fmt.Sprintf("git push --change %s", revision.Commit.GetChangeId()),
				desc:     fmt.Sprintf("Push the current change (%s)", revision.Commit.GetChangeId()),
				command:  jj.GitPushCommandArgs{Change: revision},
			},
		)
	}
	items = append(items,
		item{name: "git push --deleted", desc: "Push all deleted bookmarks", command: jj.GitPushCommandArgs{Deleted: true}, category: itemCategoryPush, key: "d"},
		item{name: "git push --tracked", desc: "Push all tracked bookmarks (including deleted bookmarks)", command: jj.GitPushCommandArgs{Tracked: true}, category: itemCategoryPush, key: "t"},
		item{name: "git push --allow-new", desc: "Allow pushing new bookmarks", command: jj.GitPushCommandArgs{AllowNew: true}, category: itemCategoryPush},
		item{name: "git fetch", desc: "Fetch from remote", command: jj.GitFetchArgs{}, category: itemCategoryFetch, key: "f"},
		item{name: "git fetch --all-remotes", desc: "Fetch from all remotes", command: jj.GitFetchArgs{AllRemotes: true}, category: itemCategoryFetch, key: "a"},
	)

	size := view.NewSizeable(0, 0)
	keymap := config.Current.GetKeyMap()
	menu := menu.NewMenu(items, size.Width, size.Height, keymap, menu.WithStylePrefix("git"))
	menu.Title = "Git Operations"
	menu.FilterMatches = func(i list.Item, filter string) bool {
		if gitItem, ok := i.(item); ok {
			return gitItem.category == itemCategory(filter)
		}
		return false
	}

	m := &Model{
		context: c,
		menu:    menu,
		keymap:  keymap,
	}
	return m
}
