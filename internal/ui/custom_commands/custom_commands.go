package customcommands

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/menu"
	"github.com/idursun/jjui/internal/ui/context"
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

var _ common.Model = (*Model)(nil)

type Model struct {
	context *context.MainContext
	keymap  config.KeyMappings[key.Binding]
	menu    menu.Menu
	help    help.Model
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keymap.Cancel,
		m.keymap.Apply,
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

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.menu.List.SettingFilter() {
			break
		}
		switch {
		case key.Matches(msg, m.keymap.Apply):
			if item, ok := m.menu.List.SelectedItem().(item); ok {
				return tea.Batch(item.command, common.Close)
			}
		case key.Matches(msg, m.keymap.Cancel):
			if m.menu.Filter != "" || m.menu.List.IsFiltered() {
				m.menu.List.ResetFilter()
				return m.menu.Filtered("")
			}
			return common.Close
		default:
			for _, listItem := range m.menu.List.Items() {
				if i, ok := listItem.(item); ok && key.Matches(msg, i.key) {
					return tea.Batch(i.command, common.Close)
				}
			}
		}
	}
	var cmd tea.Cmd
	m.menu.List, cmd = m.menu.List.Update(msg)
	return cmd
}

func (m *Model) View() string {
	return m.menu.View()
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
	menu := menu.NewMenu(items, width, height, keyMap, menu.WithStylePrefix("custom_commands"))
	menu.Title = "Custom Commands"
	menu.ShowShortcuts(true)
	menu.FilterMatches = func(i list.Item, filter string) bool {
		return strings.Contains(strings.ToLower(i.FilterValue()), strings.ToLower(filter))
	}

	m := &Model{
		context: ctx,
		keymap:  keyMap,
		menu:    menu,
		help:    help.New(),
	}
	m.SetWidth(width)
	m.SetHeight(height)
	return m
}
