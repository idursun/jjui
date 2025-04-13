package customcommands

import (
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
	command InvokableCustomCommand
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
	context        context.AppContext
	commandManager *CommandManager
	keymap         config.KeyMappings[key.Binding]
	list           list.Model
	width          int
	height         int
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
		switch {
		case key.Matches(msg, m.keymap.Apply):
			if item, ok := m.list.SelectedItem().(item); ok {
				return m, tea.Batch(item.command.Invoke(m.context), common.Close)
			}
		case key.Matches(msg, m.keymap.Cancel):
			return m, common.Close
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	content := lipgloss.Place(m.width, m.height, 0, 0, m.list.View())
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Render(content)
}

func NewModel(ctx context.AppContext, width int, height int) *Model {
	var items []list.Item

	commandManager := InitCommandManager()
	for command := range commandManager.Iter(ctx) {
		invokableCmd := command.Prepare(ctx)
		items = append(items, item{name: command.name, desc: strings.Join(invokableCmd.args, " "), command: invokableCmd})
	}
	keyMap := ctx.KeyMap()
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.SetShowTitle(true)
	l.Title = "Custom Commands"
	l.SetShowTitle(true)
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	m := &Model{
		context:        ctx,
		commandManager: commandManager,
		keymap:         keyMap,
		list:           l,
	}
	m.SetWidth(width)
	m.SetHeight(height)
	return m
}
