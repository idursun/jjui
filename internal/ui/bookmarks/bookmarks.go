package bookmarks

import (
	"fmt"
	"math"
	"slices"
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

type updateItemsMsg struct {
	items []menu.Item
}

// SelectRemoteMsg is sent when a remote is clicked
type SelectRemoteMsg struct {
	Index int
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
	current           *jj.Commit
	menu              menu.Menu
	keymap            config.KeyMappings[key.Binding]
	distanceMap       map[string]int
	remoteNames       []string
	selectedRemoteIdx int
	styles            styles
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
		m.menu.FilterKey,
		key.NewBinding(
			key.WithKeys("tab/shift+tab"),
			key.WithHelp("tab/shift+tab", "cycle remotes")),
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
	args     []string
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
	desc := strings.Join(i.args, " ")
	return desc
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.loadAll, m.loadMovables)
}

func (m *Model) filtered(filter string) tea.Cmd {
	return m.menu.Filtered(filter)
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

	if m.menu.Filter != "" {
		return m.menu.Filtered(m.menu.Filter)
	}
	return m.menu.SetItems(m.menu.Items)
}

func (m *Model) loadMovables() tea.Msg {
	output, _ := m.context.RunCommandImmediate(jj.BookmarkListMovable(m.current.GetChangeId()))
	var bookmarkItems []menu.Item
	bookmarks := jj.ParseBookmarkListOutput(string(output))
	for _, b := range bookmarks {
		if !b.Conflict && b.CommitId == m.current.CommitId {
			continue
		}

		name := fmt.Sprintf("move '%s' to %s", b.Name, m.current.GetChangeId())
		if b.Conflict {
			name = fmt.Sprintf("move conflicted '%s' to %s", b.Name, m.current.GetChangeId())
		}
		var extraFlags []string
		if b.Backwards {
			name = fmt.Sprintf("move '%s' backwards to %s", b.Name, m.current.GetChangeId())
			extraFlags = append(extraFlags, "--allow-backwards")
		}
		elem := item{
			name:     name,
			priority: moveCommand,
			args:     jj.BookmarkMove(m.current.GetChangeId(), b.Name, extraFlags...),
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
	if output, err := m.context.RunCommandImmediate(jj.BookmarkListAll()); err != nil {
		return nil
	} else {
		bookmarks := jj.ParseBookmarkListOutput(string(output))

		items := make([]menu.Item, 0)
		for _, b := range bookmarks {
			distance := m.distance(b.CommitId)
			if b.IsDeletable() {
				items = append(items, item{
					name:     fmt.Sprintf("delete '%s'", b.Name),
					priority: deleteCommand,
					dist:     distance,
					args:     jj.BookmarkDelete(b.Name),
				})
			}

			items = append(items, item{
				name:     fmt.Sprintf("forget '%s'", b.Name),
				priority: forgetCommand,
				dist:     distance,
				args:     jj.BookmarkForget(b.Name),
			})

			// Track local bookmarks as they have no remotes
			if b.IsTrackable() {
				items = append(items, item{
					name:     fmt.Sprintf("track '%s'", b.Name),
					priority: trackCommand,
					dist:     distance,
					args:     jj.BookmarkTrack(b.Name),
				})
			}

			for _, remote := range b.Remotes {
				nameWithRemote := fmt.Sprintf("%s@%s", b.Name, remote.Remote)
				if remote.Tracked {
					items = append(items, item{
						name:     fmt.Sprintf("untrack '%s'", nameWithRemote),
						priority: untrackCommand,
						dist:     distance,
						args:     jj.BookmarkUntrack(b.Name, remote.Remote),
					})
				} else {
					items = append(items, item{
						name:     fmt.Sprintf("track '%s'", nameWithRemote),
						priority: trackCommand,
						dist:     distance,
						args:     jj.BookmarkTrack(nameWithRemote),
					})
				}
			}
		}
		return updateItemsMsg{items: items}
	}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case SelectRemoteMsg:
		if msg.Index >= 0 && msg.Index < len(m.remoteNames) {
			m.selectedRemoteIdx = msg.Index
			if m.menu.Filter != "" {
				return m.menu.Filtered(m.menu.Filter)
			}
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
		case key.Matches(msg, m.keymap.Cancel):
			if m.menu.Filter != "" || m.menu.IsFiltered() {
				m.menu.ResetFilter()
				return m.filtered("")
			}
			return common.Close
		case key.Matches(msg, m.keymap.Apply):
			if m.menu.SelectedItem() == nil {
				break
			}
			action := m.menu.SelectedItem().(item)
			return m.context.RunCommand(action.args, common.Refresh, common.Close)
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
			for _, listItem := range m.menu.VisibleItems() {
				if item, ok := listItem.(item); ok && m.menu.Filter != "" && item.key == msg.String() {
					return m.context.RunCommand(jj.Args(item.args...), common.Refresh, common.Close)
				}
			}
		}
	case updateItemsMsg:
		m.menu.Items = append(m.menu.Items, msg.items...)
		slices.SortFunc(m.menu.Items, itemSorter)
		return m.menu.SetItems(m.menu.Items)
	}
	return m.menu.Update(msg)
}

func itemSorter(a menu.Item, b menu.Item) int {
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
		dl.AddFill(fillRect, ' ', lipgloss.NewStyle(), 1)
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
	// Create a window for remotes with higher z-index than menu (z=10)
	// so that clicks are routed to this window instead of the menu
	remoteRect := cellbuf.Rect(x, y, width, 1)
	windowedDl := dl.Window(remoteRect, 11)

	// Use z=2 to render above menu content (menu uses z=0 for border, z=1 for content)
	tb := windowedDl.Text(x, y, 2).
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

func (m *Model) distance(commitId string) int {
	if dist, ok := m.distanceMap[commitId]; ok {
		return dist
	}
	return math.MinInt32
}

func loadRemoteNames(c context.CommandRunner) []string {
	bytes, _ := c.RunCommandImmediate(jj.GitRemoteList())
	remotes := jj.ParseRemoteListOutput(string(bytes))
	return remotes
}

func NewModel(c *context.MainContext, current *jj.Commit, commitIds []string) *Model {
	var items []menu.Item
	keymap := config.Current.GetKeyMap()
	remotes := loadRemoteNames(c)

	styles := styles{
		promptStyle:   common.DefaultPalette.Get("title"),
		textStyle:     common.DefaultPalette.Get("dimmed"),
		selectedStyle: common.DefaultPalette.Get("menu selected"),
		noRemoteStyle: common.DefaultPalette.Get("error"),
	}

	menuModel := menu.NewMenu(items, keymap, menu.WithStylePrefix("bookmarks"))
	menuModel.Title = "Bookmark Operations"
	menuModel.Subtitle = " " // placeholder to reserve space; actual remotes rendered via TextBuilder

	m := &Model{
		context:           c,
		keymap:            keymap,
		menu:              menuModel,
		current:           current,
		distanceMap:       calcDistanceMap(current.CommitId, commitIds),
		remoteNames:       remotes,
		selectedRemoteIdx: 0,
		styles:            styles,
	}

	// Set FilterMatches after m is created so the closure can reference m
	m.menu.FilterMatches = func(i menu.Item, filter string) bool {
		if !strings.HasPrefix(i.FilterValue(), filter) {
			return false
		}
		// If filtering track/untrack and a remote is selected, filter by remote
		if len(m.remoteNames) > 0 && m.selectedRemoteIdx < len(m.remoteNames) {
			selectedRemote := m.remoteNames[m.selectedRemoteIdx]
			if strings.HasPrefix(filter, "track") || strings.HasPrefix(filter, "untrack") {
				// Only show items that contain @selectedRemote
				if strings.Contains(i.FilterValue(), "@") {
					return strings.Contains(i.FilterValue(), "@"+selectedRemote)
				}
			}
		}
		return true
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
