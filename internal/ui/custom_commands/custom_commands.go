package customcommands

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"strings"
)

type item struct {
	name    string
	desc    string
	command tea.Cmd
	key     key.Binding
}

func (i item) ShortCut() string {
	k := strings.Join(i.key.Keys(), "/")
	return k
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
	width   int
	height  int
	help    help.Model
}

func (m *Model) Width() int {
	return m.width
}

func (m *Model) Height() int {
	return m.height
}

func (m *Model) SetWidth(w int) {
	m.width = w
	m.list.SetWidth(m.width)
}

func (m *Model) SetHeight(h int) {
	maxHeight, minHeight := 30, 10
	m.height = max(min(maxHeight, h-4), minHeight)
	m.list.SetHeight(m.height - 4)
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
			if item, ok := m.list.SelectedItem().(item); ok {
				return m, tea.Batch(item.command, common.Close)
			}
		case key.Matches(msg, m.keymap.Cancel):
			if m.list.IsFiltered() {
				m.list.ResetFilter()
				return m, nil
			}
			return m, common.Close
		default:
			for _, listItem := range m.list.Items() {
				if i, ok := listItem.(item); ok && key.Matches(msg, i.key) {
					return m, tea.Batch(i.command, common.Close)
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
	listView := m.list.View()
	helpView := m.help.ShortHelpView([]key.Binding{m.keymap.Apply, m.keymap.Cancel, m.list.KeyMap.Filter})
	content := lipgloss.JoinVertical(0, title, "", listView, " "+helpView)
	content = lipgloss.Place(m.width, m.height, 0, 0, content)
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Render(content)
}

func NewModel(ctx *context.MainContext, width int, height int) *Model {
	var items []list.Item

	for name, command := range ctx.CustomCommands {
		if command.IsApplicableTo(ctx.SelectedItem) {
			cmd := command.Prepare(ctx)
			items = append(items, item{name: name, desc: command.Description(ctx), command: cmd, key: command.Binding()})
		}
	}
	keyMap := config.Current.GetKeyMap()
	l := list.New(items, common.ListItemDelegate{ShowShortcuts: true}, width, height)
	l.Title = "Custom Commands"
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowFilter(true)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()

	h := help.New()
	h.Styles.ShortKey = common.DefaultPalette.Shortcut
	h.Styles.ShortDesc = common.DefaultPalette.Dimmed

	m := &Model{
		context: ctx,
		keymap:  keyMap,
		help:    h,
		list:    l,
	}
	m.SetWidth(width)
	m.SetHeight(height)
	return m
}
