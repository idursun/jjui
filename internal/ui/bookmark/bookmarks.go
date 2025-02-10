package bookmark

import (
	"fmt"
	"io"
	"strings"

	"github.com/idursun/jjui/internal/ui/common"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	revision string
	list     list.Model
	op       common.Operation
	common.UICommands
}

type item string

func (b item) Title() string       { return string(b) }
func (b item) Description() string { return "" }
func (b item) FilterValue() string { return string(b) }

type itemDelegate struct{}

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
)

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}
	str := fmt.Sprintf("%d %s", index+1, i)
	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}
	fmt.Fprint(w, fn(str))
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, common.Close
		case "enter":
			bookmark := m.list.SelectedItem().(item)
			switch m.op {
			case common.MoveBookmarkOperation:
				return m, m.MoveBookmark(m.revision, string(bookmark))
			case common.DeleteBookmarkOperation:
				return m, m.DeleteBookmark(m.revision, string(bookmark))
			}
		}
	case common.UpdateBookmarksMsg:
		items := convertToItems(msg.Bookmarks)
		m.list.SetItems(items)
		m.list.SetHeight(max(0, min(10, len(items))))
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func convertToItems(bookmarks []string) []list.Item {
	var items []list.Item
	for _, bookmark := range bookmarks {
		item := item(bookmark)
		items = append(items, item)
	}
	return items
}

func (m Model) View() string {
	return m.list.View()
}

func New(commands common.UICommands, revision string, width int) tea.Model {
	l := list.New(nil, itemDelegate{}, width, 0)
	l.SetFilteringEnabled(true)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	return Model{
		revision:   revision,
		op:         common.MoveBookmarkOperation,
		list:       l,
		UICommands: commands,
	}
}

func NewDeleteBookmark(commands common.UICommands, revision string, bookmarks []string, width int) tea.Model {
	items := convertToItems(bookmarks)
	l := list.New(items, itemDelegate{}, width, max(0, min(10, len(items))))
	l.SetFilteringEnabled(true)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	return Model{
		revision:   revision,
		op:         common.DeleteBookmarkOperation,
		list:       l,
		UICommands: commands,
	}
}
