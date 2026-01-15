package customcommands

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/menu"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
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

var _ common.ImmediateModel = (*Model)(nil)

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
	context *context.MainContext
	keymap  config.KeyMappings[key.Binding]
	menu    menu.Menu
	help    help.Model
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keymap.Cancel,
		m.keymap.Apply,
		m.menu.FilterKey,
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
		if m.menu.SettingFilter() {
			break
		}
		switch {
		case key.Matches(msg, m.keymap.Apply):
			if item, ok := m.menu.SelectedItem().(item); ok {
				return tea.Batch(item.command, common.Close)
			}
		case key.Matches(msg, m.keymap.Cancel):
			if m.menu.Filter != "" || m.menu.IsFiltered() {
				m.menu.ResetFilter()
				return m.menu.Filtered("")
			}
			return common.Close
		default:
			for _, listItem := range m.menu.VisibleItems() {
				if i, ok := listItem.(item); ok && key.Matches(msg, i.key) {
					return tea.Batch(i.command, common.Close)
				}
			}
		}
	}
	return m.menu.Update(msg)
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	pw, ph := box.R.Dx(), box.R.Dy()
	contentRect := cellbuf.Rect(0, 0, min(pw, 80), min(ph, 40)).Inset(2)
	menuWidth := max(contentRect.Dx()+2, 0)
	menuHeight := max(contentRect.Dy()+2, 0)
	sx := box.R.Min.X + max((pw-menuWidth)/2, 0)
	sy := box.R.Min.Y + max((ph-menuHeight)/2, 0)
	frame := cellbuf.Rect(sx, sy, menuWidth, menuHeight)
	m.menu.ViewRect(dl, layout.Box{R: frame})
}

func NewModel(ctx *context.MainContext) *Model {
	var items []menu.Item

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

	sort.Slice(items, func(i, j int) bool {
		return items[i].(item).name < items[j].(item).name
	})

	keyMap := config.Current.GetKeyMap()
	menuModel := menu.NewMenu(items, keyMap, menu.WithStylePrefix("custom_commands"))
	menuModel.Title = "Custom Commands"
	menuModel.ShowShortcuts(true)
	menuModel.FilterMatches = func(i menu.Item, filter string) bool {
		return strings.Contains(strings.ToLower(i.FilterValue()), strings.ToLower(filter))
	}

	m := &Model{
		context: ctx,
		keymap:  keyMap,
		menu:    menuModel,
		help:    help.New(),
	}
	return m
}
