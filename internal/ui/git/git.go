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
	"strings"
)

var filterStyle = common.DefaultPalette.ChangeId.PaddingLeft(2)
var filterValueStyle = common.DefaultPalette.Normal.Bold(true)

type item struct {
	key.Binding
	name    string
	desc    string
	command []string
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
	context context.AppContext
	keymap  config.KeyMappings[key.Binding]
	list    list.Model
	items   []list.Item
	filter  string
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
		case key.Matches(msg, m.keymap.Git.Push):
			return m.filtered("push")
		case key.Matches(msg, m.keymap.Git.Fetch):
			return m.filtered("fetch")
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
		filterView = lipgloss.JoinHorizontal(0, filterStyle.Render("Showing only "), filterValueStyle.Render(m.filter))
	}
	listView := m.list.View()
	helpView := m.helpView()
	content := lipgloss.JoinVertical(0, title, "", filterView, listView, "", helpView)
	content = lipgloss.Place(m.width, m.height, 0, 0, content)
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

func (m *Model) filtered(filter string) (tea.Model, tea.Cmd) {
	m.filter = filter
	if m.filter == "" {
		return m, m.list.SetItems(m.items)
	}
	var filtered []list.Item
	for _, i := range m.items {
		if strings.Contains(i.FilterValue(), m.filter) {
			filtered = append(filtered, i)
		}
	}
	m.list.ResetSelected()
	return m, m.list.SetItems(filtered)
}

func loadBookmarks(c context.AppContext, changeId string) []jj.Bookmark {
	bytes, _ := c.RunCommandImmediate(jj.BookmarkList(changeId))
	bookmarks := jj.ParseBookmarkListOutput(string(bytes))
	return bookmarks
}

func NewModel(c context.AppContext, commit *jj.Commit, width int, height int) *Model {
	var items []list.Item
	if commit != nil {
		bookmarks := loadBookmarks(c, commit.GetChangeId())
		for _, b := range bookmarks {
			if b.Conflict {
				continue
			}
			for _, remote := range b.Remotes {
				bookmarkItem := item{
					name:    fmt.Sprintf("git push --bookmark %s --remote %s", b.Name, remote.Remote),
					desc:    fmt.Sprintf("Git push bookmark %s to remote %s", b.Name, remote.Remote),
					command: jj.GitPush("--bookmark", b.Name, "--remote", remote.Remote),
				}
				items = append(items, bookmarkItem)
			}
			if b.IsLocal() {
				bookmarkItem := item{
					name:    fmt.Sprintf("git push --bookmark %s --allow-new", b.Name),
					desc:    fmt.Sprintf("Git push new bookmark %s", b.Name),
					command: jj.GitPush("--bookmark", b.Name, "--allow-new"),
				}
				items = append(items, bookmarkItem)
			}
		}
	}
	items = append(items,
		item{name: "git push", desc: "Push tracking bookmarks in the current revset", command: jj.GitPush()},
		item{name: "git push --all", desc: "Push all bookmarks (including new and deleted bookmarks)", command: jj.GitPush("--all")},
		item{name: "git push --deleted", desc: "Push all deleted bookmarks", command: jj.GitPush("--deleted")},
		item{name: "git push --tracked", desc: "Push all tracked bookmarks (including deleted bookmarks)", command: jj.GitPush("--tracked")},
		item{name: "git push --allow-new", desc: "Allow pushing new bookmarks", command: jj.GitPush("--allow-new")},
		item{name: "git fetch", desc: "Fetch from remote", command: jj.GitFetch()},
		item{name: "git fetch --all-remotes", desc: "Fetch from all remotes", command: jj.GitFetch("--all-remotes")},
	)
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.SetShowTitle(true)
	l.Title = "Git Operations"
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(true)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	m := &Model{
		context: c,
		list:    l,
		items:   items,
		keymap:  c.KeyMap(),
	}
	m.SetWidth(width)
	m.SetHeight(height)
	return m
}
