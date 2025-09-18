package customcommands

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common/menu"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
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

var _ view.IViewModel = (*Model)(nil)

type Model struct {
	*view.ViewNode
	context        *context.MainContext
	keymap         config.KeyMappings[key.Binding]
	menu           *menu.Menu
	help           help.Model
	CustomCommands map[string]context.CustomCommand
}

func (m *Model) GetId() view.ViewId {
	return "custom commands"
}

func (m *Model) Mount(v *view.ViewNode) {
	m.ViewNode = v
	v.Id = m.GetId()
	maxWidth, minWidth := 80, 40
	m.Width = max(min(maxWidth, m.ViewManager.Width), minWidth)
	maxHeight, minHeight := 30, 10
	m.Height = max(min(maxHeight, m.ViewManager.Height), minHeight)
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

func (m *Model) Init() tea.Cmd {
	if output, err := config.LoadConfigFile(); err == nil {
		if registry, err := context.LoadCustomCommands(string(output)); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading custom commands: %v\n", err)
			os.Exit(1)
		} else {
			m.CustomCommands = registry
		}
	}

	var items []list.Item

	for name, command := range m.CustomCommands {
		if command.IsApplicableTo(m.context) {
			cmd := command.Prepare(m.context)
			items = append(items, item{name: name, desc: command.Description(m.context), command: cmd, key: command.Binding()})
		}
	}
	size := view.NewSizeable(80, 20)
	menu := menu.NewMenu(items, size.Width, size.Height, m.keymap, menu.WithStylePrefix("custom_commands"))
	menu.Title = "Custom Commands"
	menu.ShowShortcuts(true)
	menu.FilterMatches = func(i list.Item, filter string) bool {
		return strings.Contains(strings.ToLower(i.FilterValue()), strings.ToLower(filter))
	}
	m.menu = &menu
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
			if item, ok := m.menu.List.SelectedItem().(item); ok {
				m.ViewManager.UnregisterView(m.Id)
				return m, item.command
			}
		case key.Matches(msg, m.keymap.Cancel):
			if m.menu.Filter != "" || m.menu.List.IsFiltered() {
				m.menu.List.ResetFilter()
				return m, m.menu.Filtered("")
			}
			m.ViewManager.UnregisterView(m.Id)
			return m, nil
		default:
			for _, listItem := range m.menu.List.Items() {
				if i, ok := listItem.(item); ok && key.Matches(msg, i.key) {
					m.ViewManager.UnregisterView(m.Id)
					return m, i.command
				}
			}
		}
	}
	var cmd tea.Cmd
	m.menu.List, cmd = m.menu.List.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	if m.menu != nil {
		return m.menu.View()
	}
	return ""
}

func NewModel(ctx *context.MainContext) *Model {
	keyMap := config.Current.GetKeyMap()

	m := &Model{
		context: ctx,
		keymap:  keyMap,
		help:    help.New(),
	}
	return m
}
