package customcommands

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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
	context        *context.MainContext
	keymap         config.KeyMappings[key.Binding]
	filterableList common.FilterableList
	help           help.Model
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
			if item, ok := m.filterableList.List.SelectedItem().(item); ok {
				return m, tea.Batch(item.command, common.Close)
			}
		case key.Matches(msg, m.keymap.Cancel):
			if m.filterableList.Filter != "" || m.filterableList.List.IsFiltered() {
				m.filterableList.List.ResetFilter()
				return m, m.filterableList.Filtered("")
			}
			return m, common.Close
		default:
			for _, listItem := range m.filterableList.List.Items() {
				if i, ok := listItem.(item); ok && key.Matches(msg, i.key) {
					return m, tea.Batch(i.command, common.Close)
				}
			}
		}
	}
	var cmd tea.Cmd
	m.filterableList.List, cmd = m.filterableList.List.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	return m.filterableList.View(nil)
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
	filterableList := common.NewFilterableList(items, width, height, keyMap)
	filterableList.Title = "Custom Commands"
	filterableList.ShowShortcuts(true)
	filterableList.FilterMatches = func(i list.Item, filter string) bool {
		return strings.Contains(strings.ToLower(i.FilterValue()), strings.ToLower(filter))
	}

	m := &Model{
		context:        ctx,
		keymap:         keyMap,
		filterableList: filterableList,
		help:           help.New(),
	}
	m.SetWidth(width)
	m.SetHeight(height)
	return m
}
