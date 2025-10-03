package git

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/menu"
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
	context *context.MainContext
	keymap  config.KeyMappings[key.Binding]
	menu    menu.Menu
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

func (m *Model) Width() int {
	return m.menu.Width()
}

func (m *Model) Height() int {
	return m.menu.Height()
}

func (m *Model) SetWidth(w int) {
	m.menu.SetWidth(w)
}

func (m *Model) SetHeight(h int) {
	m.menu.SetHeight(h)
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
			return m, m.context.RunCommand(jj.Args(action.command...), common.Refresh, common.Close)
		case key.Matches(msg, m.keymap.Cancel):
			if m.menu.Filter != "" || m.menu.List.IsFiltered() {
				m.menu.List.ResetFilter()
				return m.filtered("")
			}
			return m, common.Close
		case key.Matches(msg, m.keymap.Git.Push) && m.menu.Filter != string(itemCategoryPush):
			return m.filtered(string(itemCategoryPush))
		case key.Matches(msg, m.keymap.Git.Fetch) && m.menu.Filter != string(itemCategoryFetch):
			return m.filtered(string(itemCategoryFetch))
		default:
			for _, listItem := range m.menu.List.Items() {
				if item, ok := listItem.(item); ok && m.menu.Filter != "" && item.key == msg.String() {
					return m, m.context.RunCommand(jj.Args(item.command...), common.Refresh, common.Close)
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

func loadBookmarks(c context.CommandRunner, changeId string) []jj.Bookmark {
	bytes, _ := c.RunCommandImmediate(jj.BookmarkList(changeId))
	bookmarks := jj.ParseBookmarkListOutput(string(bytes))
	return bookmarks
}

func NewModel(c *context.MainContext, revisions jj.SelectedRevisions, width int, height int) *Model {
	var items []list.Item
	for _, commit := range revisions.Revisions {
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

	hasMultipleRevisions := len(revisions.Revisions) > 1

	if hasMultipleRevisions {
		items = append(items,
			item{
				key:      "c",
				category: itemCategoryPush,
				name:     fmt.Sprintf("git push %s", strings.Join(revisions.AsPrefixedArgs("--change"), " ")),
				desc:     fmt.Sprintf("Push selected changes (%s)", strings.Join(revisions.GetIds(), " ")),
				command:  jj.GitPush(revisions.AsPrefixedArgs("--change")...),
			})
	}

	for _, commit := range revisions.Revisions {
		item := item{
			category: itemCategoryPush,
			name:     fmt.Sprintf("git push --change %s", commit.GetChangeId()),
			desc:     fmt.Sprintf("Push the current change (%s)", commit.GetChangeId()),
			command:  jj.GitPush("--change", commit.GetChangeId()),
		}

		if !hasMultipleRevisions {
			item.key = "c"
		}
		items = append(items, item)
	}

	items = append(items,
		item{name: "git push --deleted", desc: "Push all deleted bookmarks", command: jj.GitPush("--deleted"), category: itemCategoryPush, key: "d"},
		item{name: "git push --tracked", desc: "Push all tracked bookmarks (including deleted bookmarks)", command: jj.GitPush("--tracked"), category: itemCategoryPush, key: "t"},
		item{name: "git push --allow-new", desc: "Allow pushing new bookmarks", command: jj.GitPush("--allow-new"), category: itemCategoryPush},
		item{name: "git fetch", desc: "Fetch from remote", command: jj.GitFetch(), category: itemCategoryFetch, key: "f"},
		item{name: "git fetch --all-remotes", desc: "Fetch from all remotes", command: jj.GitFetch("--all-remotes"), category: itemCategoryFetch, key: "a"},
	)

	keymap := config.Current.GetKeyMap()
	menu := menu.NewMenu(items, width, height, keymap, menu.WithStylePrefix("git"))
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
	m.SetWidth(width)
	m.SetHeight(height)
	return m
}
