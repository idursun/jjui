package git

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
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

type Model struct {
	context        *context.MainContext
	keymap         config.KeyMappings[key.Binding]
	filterableList common.FilterableList
}

func (m *Model) Width() int {
	return m.filterableList.Width
}

func (m *Model) Height() int {
	return m.filterableList.Height
}

func (m *Model) SetWidth(w int) {
	m.filterableList.SetWidth(w)
}

func (m *Model) SetHeight(h int) {
	m.filterableList.SetHeight(h)
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filterableList.List.SettingFilter() {
			break
		}
		switch {
		case key.Matches(msg, m.keymap.Apply):
			action := m.filterableList.List.SelectedItem().(item)
			return m, m.context.RunCommand(jj.Args(action.command...), common.Refresh, common.Close)
		case key.Matches(msg, m.keymap.Cancel):
			if m.filterableList.Filter != "" || m.filterableList.List.IsFiltered() {
				m.filterableList.List.ResetFilter()
				return m.filtered("")
			}
			return m, common.Close
		case key.Matches(msg, m.keymap.Git.Push) && m.filterableList.Filter != string(itemCategoryPush):
			return m.filtered(string(itemCategoryPush))
		case key.Matches(msg, m.keymap.Git.Fetch) && m.filterableList.Filter != string(itemCategoryFetch):
			return m.filtered(string(itemCategoryFetch))
		default:
			for _, listItem := range m.filterableList.List.Items() {
				if item, ok := listItem.(item); ok && m.filterableList.Filter != "" && item.key == msg.String() {
					return m, m.context.RunCommand(jj.Args(item.command...), common.Refresh, common.Close)
				}
			}
		}
	}
	var cmd tea.Cmd
	m.filterableList.List, cmd = m.filterableList.List.Update(msg)
	return m, cmd
}

func (m *Model) filtered(filter string) (tea.Model, tea.Cmd) {
	return m, m.filterableList.Filtered(filter)
}

func (m *Model) View() string {
	helpKeys := []key.Binding{
		m.keymap.Git.Push,
		m.keymap.Git.Fetch,
	}

	return m.filterableList.View(helpKeys)
}

func loadBookmarks(c context.CommandRunner, changeId string) []jj.Bookmark {
	bytes, _ := c.RunCommandImmediate(jj.BookmarkList(changeId))
	bookmarks := jj.ParseBookmarkListOutput(string(bytes))
	return bookmarks
}

func NewModel(c *context.MainContext, commit *jj.Commit, width int, height int) *Model {
	var items []list.Item
	if commit != nil {
		bookmarks := loadBookmarks(c, commit.GetChangeId())
		for _, b := range bookmarks {
			if b.Conflict {
				continue
			}
			for _, remote := range b.Remotes {
				items = append(items, item{
					name:     fmt.Sprintf("git push --bookmark %s --remote %s", b.Name, remote.Remote),
					desc:     fmt.Sprintf("Git push bookmark %s to remote %s", b.Name, remote.Remote),
					command:  jj.GitPush("--bookmark", b.Name, "--remote", remote.Remote),
					category: itemCategoryPush,
				})
			}
			if b.IsPushable() {
				items = append(items, item{
					name:     fmt.Sprintf("git push --bookmark %s --allow-new", b.Name),
					desc:     fmt.Sprintf("Git push new bookmark %s", b.Name),
					command:  jj.GitPush("--bookmark", b.Name, "--allow-new"),
					category: itemCategoryPush,
				})
			}
		}
	}
	items = append(items,
		item{name: "git push", desc: "Push tracking bookmarks in the current revset", command: jj.GitPush(), category: itemCategoryPush, key: "p"},
		item{name: "git push --all", desc: "Push all bookmarks (including new and deleted bookmarks)", command: jj.GitPush("--all"), category: itemCategoryPush, key: "a"},
	)
	if commit != nil {
		items = append(items,
			item{
				key:      "c",
				category: itemCategoryPush,
				name:     fmt.Sprintf("git push --change %s", commit.GetChangeId()),
				desc:     fmt.Sprintf("Push the current change (%s)", commit.GetChangeId()),
				command:  jj.GitPush("--change", commit.GetChangeId()),
			},
		)
	}
	items = append(items,
		item{name: "git push --deleted", desc: "Push all deleted bookmarks", command: jj.GitPush("--deleted"), category: itemCategoryPush, key: "d"},
		item{name: "git push --tracked", desc: "Push all tracked bookmarks (including deleted bookmarks)", command: jj.GitPush("--tracked"), category: itemCategoryPush},
		item{name: "git push --allow-new", desc: "Allow pushing new bookmarks", command: jj.GitPush("--allow-new"), category: itemCategoryPush},
		item{name: "git fetch", desc: "Fetch from remote", command: jj.GitFetch(), category: itemCategoryFetch, key: "f"},
		item{name: "git fetch --all-remotes", desc: "Fetch from all remotes", command: jj.GitFetch("--all-remotes"), category: itemCategoryFetch, key: "a"},
	)

	keymap := config.Current.GetKeyMap()
	filterableList := common.NewFilterableList(items, width, height, keymap)
	filterableList.Title = "Git Operations"
	filterableList.FilterMatches = func(i list.Item, filter string) bool {
		if gitItem, ok := i.(item); ok {
			return gitItem.category == itemCategory(filter)
		}
		return false
	}

	m := &Model{
		context:        c,
		filterableList: filterableList,
		keymap:         keymap,
	}
	m.SetWidth(width)
	m.SetHeight(height)
	return m
}
