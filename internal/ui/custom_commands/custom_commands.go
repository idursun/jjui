package customcommands

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/menu"
	"github.com/idursun/jjui/internal/ui/context"
)

type item struct {
	name        string
	desc        string
	command     tea.Cmd
	key         key.Binding
	keySequence []key.Binding
}

func (i item) ShortCut() string {
	if len(i.keySequence) > 0 {
		var keys []string
		for _, k := range i.keySequence {
			keys = append(keys, k.Keys()...)
		}
		return strings.Join(keys, " → ")
	}
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

// SortedCustomCommands returns commands ordered by name for deterministic iteration.
func SortedCustomCommands(ctx *context.MainContext) []context.CustomCommand {
	names := make([]string, 0, len(ctx.CustomCommands))
	for name := range ctx.CustomCommands {
		names = append(names, name)
	}
	sort.Strings(names)

	commands := make([]context.CustomCommand, 0, len(names))
	for _, name := range names {
		commands = append(commands, ctx.CustomCommands[name])
	}
	return commands
}

type Model struct {
	*common.ViewNode
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
	pw, ph := m.Parent.Width, m.Parent.Height
	m.menu.SetFrame(cellbuf.Rect(0, 0, min(pw, 80), min(ph, 40)).Inset(2))
	v := m.menu.View()
	w, h := lipgloss.Size(v)
	sx := (pw - w) / 2
	sy := (ph - h) / 2
	m.SetFrame(cellbuf.Rect(sx, sy, w, h))
	return v
}

func NewModel(ctx *context.MainContext) *Model {
	var items []list.Item

	for name, command := range ctx.CustomCommands {
		if command.IsApplicableTo(ctx.SelectedItem) {
			cmd := command.Prepare(ctx)
			desc := command.Description(ctx)
			if lc, ok := command.(context.LabeledCommand); ok {
				desc = lc.Label()
			}
			items = append(items, item{name: name, desc: desc, command: cmd, key: command.Binding(), keySequence: command.Sequence()})
		}
	}
	keyMap := config.Current.GetKeyMap()
	menu := menu.NewMenu(items, keyMap, menu.WithStylePrefix("custom_commands"))
	menu.Title = "Custom Commands"
	menu.ShowShortcuts(true)
	menu.FilterMatches = func(i list.Item, filter string) bool {
		return strings.Contains(strings.ToLower(i.FilterValue()), strings.ToLower(filter))
	}

	m := &Model{
		ViewNode: common.NewViewNode(0, 0),
		context:  ctx,
		keymap:   keyMap,
		menu:     menu,
		help:     help.New(),
	}
	menu.Parent = m.ViewNode
	return m
}
