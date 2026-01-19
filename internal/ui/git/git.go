package git

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/menu"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type itemCategory string

const (
	itemCategoryPush  itemCategory = "push"
	itemCategoryFetch itemCategory = "fetch"
)

// SelectRemoteMsg is sent when a remote is clicked
type SelectRemoteMsg struct {
	Index int
}

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

type styles struct {
	promptStyle   lipgloss.Style
	textStyle     lipgloss.Style
	selectedStyle lipgloss.Style
	noRemoteStyle lipgloss.Style
}

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	context           *context.MainContext
	keymap            config.KeyMappings[key.Binding]
	menu              menu.Menu
	revisions         jj.SelectedRevisions
	remoteNames       []string
	selectedRemoteIdx int
	styles            styles
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keymap.Cancel,
		m.keymap.Apply,
		m.keymap.Git.Push,
		m.keymap.Git.Fetch,
		m.menu.FilterKey,
		key.NewBinding(
			key.WithKeys("tab/shift+tab"),
			key.WithHelp("tab/shift+tab", "cycle remotes")),
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) cycleRemotes(step int) tea.Cmd {
	if len(m.remoteNames) == 0 {
		return nil
	}

	m.selectedRemoteIdx += step
	if m.selectedRemoteIdx >= len(m.remoteNames) {
		m.selectedRemoteIdx = 0
	} else if m.selectedRemoteIdx < 0 {
		m.selectedRemoteIdx = len(m.remoteNames) - 1
	}

	// Remotes are rendered via TextBuilder in ViewRect, no need to update subtitle
	m.menu.Items = m.createMenuItems()
	if m.menu.Filter != "" {
		// NOTE: return tea.Cmd to keep the internal filter
		return m.menu.Filtered(m.menu.Filter)
	}
	return m.menu.SetItems(m.menu.Items)
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case SelectRemoteMsg:
		if msg.Index >= 0 && msg.Index < len(m.remoteNames) {
			m.selectedRemoteIdx = msg.Index
			m.menu.Items = m.createMenuItems()
			return m.menu.SetItems(m.menu.Items)
		}
		return nil
	case tea.KeyMsg:
		if m.menu.SettingFilter() {
			break
		}
		switch {
		case msg.Type == tea.KeyTab:
			return m.cycleRemotes(1)
		case msg.Type == tea.KeyShiftTab:
			return m.cycleRemotes(-1)
		case key.Matches(msg, m.keymap.Apply):
			action := m.menu.SelectedItem().(item)
			return m.context.RunCommand(jj.Args(action.command...), common.Refresh, common.Close)
		case key.Matches(msg, m.keymap.Cancel):
			if m.menu.Filter != "" || m.menu.IsFiltered() {
				m.menu.ResetFilter()
				return m.filtered("")
			}
			return common.Close
		case key.Matches(msg, m.keymap.Git.Push) && m.menu.Filter != string(itemCategoryPush):
			return m.filtered(string(itemCategoryPush))
		case key.Matches(msg, m.keymap.Git.Fetch) && m.menu.Filter != string(itemCategoryFetch):
			return m.filtered(string(itemCategoryFetch))
		default:
			for _, listItem := range m.menu.VisibleItems() {
				if item, ok := listItem.(item); ok && m.menu.Filter != "" && item.key == msg.String() {
					return m.context.RunCommand(jj.Args(item.command...), common.Refresh, common.Close)
				}
			}
		}
	}
	return m.menu.Update(msg)
}

func (m *Model) filtered(filter string) tea.Cmd {
	return m.menu.Filtered(filter)
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	pw, ph := box.R.Dx(), box.R.Dy()
	contentRect := cellbuf.Rect(0, 0, min(pw, 80), min(ph, 40)).Inset(2)
	menuWidth := max(contentRect.Dx()+2, 0)
	menuHeight := max(contentRect.Dy()+2, 0)
	sx := box.R.Min.X + max((pw-menuWidth)/2, 0)
	sy := box.R.Min.Y + max((ph-menuHeight)/2, 0)
	frame := cellbuf.Rect(sx, sy, menuWidth, menuHeight)
	if len(m.menu.VisibleItems()) == 0 {
		fillRect := frame.Inset(1)
		dl.AddFill(fillRect, ' ', lipgloss.NewStyle(), menu.ZIndexContent)
	}
	m.menu.ViewRect(dl, layout.Box{R: frame})

	// Render clickable remotes in the subtitle area
	// Position: inside the menu border, after title line, with subtitle padding
	remoteY := sy + 1 + 1 + 1 // border(1) + title(1) + subtitle top padding(1)
	remoteX := sx + 1 + 1     // border(1) + subtitle left padding(1)
	remoteWidth := menuWidth - 4
	m.renderRemotes(dl, remoteX, remoteY, remoteWidth)
}

func (m *Model) renderRemotes(dl *render.DisplayContext, x, y, width int) {
	// Create a window for remotes with higher z-index than menu
	// so that clicks are routed to this window instead of the menu
	remoteRect := cellbuf.Rect(x, y, width, menu.ZIndexContent)
	windowedDl := dl.Window(remoteRect, menu.ZIndexContent)

	// Render above menu content
	tb := windowedDl.Text(x, y, menu.ZIndexContent+1).
		Styled("Remotes: ", m.styles.promptStyle)

	if len(m.remoteNames) == 0 {
		tb.Styled("NO REMOTE FOUND", m.styles.noRemoteStyle).Done()
		return
	}

	for idx, remoteName := range m.remoteNames {
		style := m.styles.textStyle
		if idx == m.selectedRemoteIdx {
			style = m.styles.selectedStyle
		}
		tb.Clickable(remoteName, style, SelectRemoteMsg{Index: idx}).
			Write(" ")
	}

	tb.Done()
}

func (m *Model) displayRemotes() string {
	var w strings.Builder
	w.WriteString(m.styles.promptStyle.PaddingRight(1).Render("Remotes:"))
	if len(m.remoteNames) == 0 {
		w.WriteString(m.styles.noRemoteStyle.Render("NO REMOTE FOUND"))
		return w.String()
	}
	for idx, remoteName := range m.remoteNames {
		if idx == m.selectedRemoteIdx {
			w.WriteString(m.styles.selectedStyle.Render(remoteName))
		} else {
			w.WriteString(m.styles.textStyle.Render(remoteName))
		}
		w.WriteString(" ")
	}
	return w.String()
}

func loadBookmarks(c context.CommandRunner, changeId string) []jj.Bookmark {
	bytes, _ := c.RunCommandImmediate(jj.BookmarkList(changeId))
	bookmarks := jj.ParseBookmarkListOutput(string(bytes))
	return bookmarks
}

func loadRemoteNames(c context.CommandRunner) []string {
	bytes, _ := c.RunCommandImmediate(jj.GitRemoteList())
	remotes := jj.ParseRemoteListOutput(string(bytes))
	return remotes
}

func NewModel(c *context.MainContext, revisions jj.SelectedRevisions) *Model {
	remotes := loadRemoteNames(c)
	keymap := config.Current.GetKeyMap()

	styles := styles{
		promptStyle:   common.DefaultPalette.Get("title"),
		textStyle:     common.DefaultPalette.Get("dimmed"),
		selectedStyle: common.DefaultPalette.Get("menu selected"),
		noRemoteStyle: common.DefaultPalette.Get("error"),
	}

	m := &Model{
		context:           c,
		keymap:            keymap,
		revisions:         revisions,
		remoteNames:       remotes,
		selectedRemoteIdx: 0,
		styles:            styles,
	}

	items := m.createMenuItems()
	m.menu = menu.NewMenu(items, m.keymap, menu.WithStylePrefix("git"))
	m.menu.Title = "Git Operations"
	m.menu.Subtitle = " " // placeholder to reserve space; actual remotes rendered via TextBuilder
	m.menu.FilterMatches = func(i menu.Item, filter string) bool {
		if gitItem, ok := i.(item); ok {
			return gitItem.category == itemCategory(filter)
		}
		return false
	}

	return m
}

func (m *Model) createMenuItems() []menu.Item {
	revisions := m.revisions
	var items []menu.Item
	hasRemote := len(m.remoteNames) > 0
	var selectedRemote string
	if hasRemote {
		selectedRemote = m.remoteNames[m.selectedRemoteIdx]
	} else {
		// set selectedRemote to empty string and `git` command fails gracefully
		selectedRemote = ""
	}

	for _, commit := range revisions.Revisions {
		bookmarks := loadBookmarks(m.context, commit.GetChangeId())
		for _, b := range bookmarks {
			if b.Conflict {
				continue
			}
			for _, remote := range b.Remotes {
				items = append(items, item{
					name:     fmt.Sprintf("git push --bookmark %s --remote %s", b.Name, remote.Remote),
					desc:     fmt.Sprintf("Git push bookmark %s to %s", b.Name, remote.Remote),
					command:  jj.GitPush("--bookmark", b.Name, "--remote", remote.Remote),
					category: itemCategoryPush,
				})
			}
		}
	}
	items = append(items,
		item{
			name:     fmt.Sprintf("git push --remote %s", selectedRemote),
			desc:     "Push tracking bookmarks in the current revset",
			command:  jj.GitPush("--remote", selectedRemote),
			category: itemCategoryPush,
			key:      "p",
		},
		item{
			name:     fmt.Sprintf("git push --all --remote %s", selectedRemote),
			desc:     "Push all bookmarks (including new and deleted bookmarks)",
			command:  jj.GitPush("--all", "--remote", selectedRemote),
			category: itemCategoryPush,
			key:      "a",
		},
	)

	hasMultipleRevisions := len(revisions.Revisions) > 1

	if hasMultipleRevisions {
		flags := []string{"--remote", selectedRemote}
		flags = append(flags, revisions.AsPrefixedArgs("--change")...)
		items = append(items,
			item{
				key:      "c",
				category: itemCategoryPush,
				name:     fmt.Sprintf("git push %s", strings.Join(revisions.AsPrefixedArgs("--change"), " ")),
				desc:     fmt.Sprintf("Push selected changes (%s)", strings.Join(revisions.GetIds(), " ")),
				command:  jj.GitPush(flags...),
			})
	}

	for _, commit := range revisions.Revisions {
		item := item{
			category: itemCategoryPush,
			name:     fmt.Sprintf("git push --change %s --remote %s", commit.GetChangeId(), selectedRemote),
			desc:     fmt.Sprintf("Push the current change (%s)", commit.GetChangeId()),
			command:  jj.GitPush("--change", commit.GetChangeId(), "--remote", selectedRemote),
		}

		if !hasMultipleRevisions {
			item.key = "c"
		}
		items = append(items, item)
	}

	items = append(items,
		item{
			name:     fmt.Sprintf("git push --deleted --remote %s", selectedRemote),
			desc:     "Push all deleted bookmarks",
			command:  jj.GitPush("--deleted", "--remote", selectedRemote),
			category: itemCategoryPush,
			key:      "d",
		},
		item{
			name:     fmt.Sprintf("git push --tracked --remote %s", selectedRemote),
			desc:     "Push all tracked bookmarks (including deleted bookmarks)",
			command:  jj.GitPush("--tracked", "--remote", selectedRemote),
			category: itemCategoryPush,
			key:      "t",
		},
		item{
			name:     fmt.Sprintf("git fetch --remote %s", selectedRemote),
			desc:     "Fetch from remote",
			command:  jj.GitFetch("--remote", selectedRemote),
			category: itemCategoryFetch, key: "f",
		},
		item{
			name:     fmt.Sprintf("git fetch --tracked --remote %s", selectedRemote),
			desc:     "Fetch from remote",
			command:  jj.GitFetch("--tracked", "--remote", selectedRemote),
			category: itemCategoryFetch, key: "f",
		},
		item{
			name:     "git fetch --all-remotes",
			desc:     "Fetch from all remotes",
			command:  jj.GitFetch("--all-remotes"),
			category: itemCategoryFetch,
			key:      "a",
		},
	)

	return items
}
