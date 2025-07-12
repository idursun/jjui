package git

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
)

var filterStyle = common.DefaultPalette.Shortcut.PaddingLeft(2)
var filterValueStyle = common.DefaultPalette.Normal.Bold(true)

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
	list    list.Model
	items   []list.Item
	filter  itemCategory
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
	//maxWidth, minWidth := 80, 40
	//m.width = max(min(maxWidth, w-4), minWidth)
	//m.list.SetWidth(m.width - 8)
	m.width = w
	m.list.SetWidth(w)
}

func (m *Model) SetHeight(h int) {
	maxHeight, minHeight := 30, 10
	m.height = max(min(maxHeight, h-4), minHeight)
	m.list.SetHeight(m.height - 6)
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.SettingFilter() {
			break
		}
		switch {
		case key.Matches(msg, m.keymap.Apply):
			action := m.list.SelectedItem().(item)
			return m, m.context.RunCommand(jj.Args(action.command...), common.Refresh, common.Close)
		case key.Matches(msg, m.keymap.Cancel):
			if m.filter != "" || m.list.IsFiltered() {
				m.list.ResetFilter()
				return m.filtered("")
			}
			return m, common.Close
		case key.Matches(msg, m.keymap.Git.Push) && m.filter != itemCategoryPush:
			return m.filtered(itemCategoryPush)
		case key.Matches(msg, m.keymap.Git.Fetch) && m.filter != itemCategoryFetch:
			return m.filtered(itemCategoryFetch)
		default:
			for _, listItem := range m.list.Items() {
				if item, ok := listItem.(item); ok && m.filter != "" && item.key == msg.String() {
					return m, m.context.RunCommand(jj.Args(item.command...), common.Refresh, common.Close)
				}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	title := m.list.Styles.Title.Render(m.list.Title)
	filterView := lipgloss.JoinHorizontal(0, filterStyle.Render("Showing "), filterValueStyle.Render("all"))
	if m.filter != "" {
		filterView = lipgloss.JoinHorizontal(0, filterStyle.Render("Showing only "), filterValueStyle.Render(string(m.filter)))
	}
	listView := m.list.View()
	helpView := m.helpView()
	content := lipgloss.JoinVertical(0, title, "", filterView, listView, "", helpView)
	content = lipgloss.Place(m.width, m.height, 0, 0, content)
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0).Render(content)
}

func renderKey(k key.Binding) string {
	if !k.Enabled() {
		return ""
	}
	return lipgloss.JoinHorizontal(0, common.DefaultPalette.Shortcut.Render(k.Help().Key, ""), common.DefaultPalette.Dimmed.Render(k.Help().Desc, ""))
}

func (m *Model) helpView() string {
	if m.list.SettingFilter() {
		return ""
	}
	bindings := []string{
		renderKey(m.keymap.Git.Push),
		renderKey(m.keymap.Git.Fetch),
	}
	if m.list.IsFiltered() {
		bindings = append(bindings, renderKey(m.keymap.Cancel))
	} else {
		bindings = append(bindings, renderKey(m.list.KeyMap.Filter))
	}

	return " " + lipgloss.JoinHorizontal(0, bindings...)
}

func (m *Model) filtered(filter itemCategory) (tea.Model, tea.Cmd) {
	m.filter = filter
	if m.filter == "" {
		m.list.SetDelegate(common.ListItemDelegate{ShowShortcuts: false})
		return m, m.list.SetItems(m.items)
	}
	m.list.SetDelegate(common.ListItemDelegate{ShowShortcuts: true})
	var filtered []list.Item
	for _, i := range m.items {
		if item, ok := i.(item); !ok || item.category != m.filter {
			continue
		}
		filtered = append(filtered, i)
	}
	m.list.ResetSelected()
	return m, m.list.SetItems(filtered)
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

	l := list.New(items, common.ListItemDelegate{}, width, height)
	l.SetShowTitle(true)
	l.Title = "Git Operations"
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(true)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	l.Styles.NoItems = common.DefaultPalette.Dimmed

	m := &Model{
		context: c,
		list:    l,
		items:   items,
		keymap:  config.Current.GetKeyMap(),
	}
	m.SetWidth(width)
	m.SetHeight(height)
	return m
}
