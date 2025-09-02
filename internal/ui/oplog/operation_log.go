package oplog

import (
	"bytes"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/common/models"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/graph"
)

type updateOpLogMsg struct {
	Rows []*models.OperationLogItem
}

type Model struct {
	*common.Sizeable
	*list.List[*models.OperationLogItem]
	context   *context.MainContext
	w         *graph.Renderer
	keymap    config.KeyMappings[key.Binding]
	textStyle lipgloss.Style
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{m.keymap.Up, m.keymap.Down, m.keymap.Cancel, m.keymap.Diff, m.keymap.OpLog.Restore}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
	return m.load()
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateOpLogMsg:
		m.Items = msg.Rows
		m.Cursor = 0
		m.w.Reset()
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			return m, common.Close
		case key.Matches(msg, m.keymap.Up):
			if m.Cursor > 0 {
				m.Cursor--
			}
		case key.Matches(msg, m.keymap.Down):
			if m.Cursor < len(m.Items)-1 {
				m.Cursor++
			}
		case key.Matches(msg, m.keymap.Diff):
			return m, func() tea.Msg {
				output, _ := m.context.RunCommandImmediate(jj.OpShow(m.Current().OperationId))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, m.keymap.OpLog.Restore):
			return m, tea.Batch(common.Close, m.context.RunCommand(jj.OpRestore(m.Current().OperationId), common.Refresh))
		}
	}
	return m, m.updateSelection()
}

func (m *Model) updateSelection() tea.Cmd {
	if m.Items == nil {
		return nil
	}
	current := m.Current()
	if current != nil {
		return m.context.SetSelectedItem(context.SelectedOperation{OperationId: current.OperationId})
	}
	return nil
}

func (m *Model) View() string {
	if m.Items == nil {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, "loading")
	}

	m.w.Reset()
	m.w.SetSize(m.Width, m.Height)
	renderer := newIterator(m.Items, m.Cursor, m.Width)
	content := m.w.Render(renderer)
	content = lipgloss.PlaceHorizontal(m.Width, lipgloss.Left, content)
	return m.textStyle.MaxWidth(m.Width).Render(content)
}

func (m *Model) load() tea.Cmd {
	return func() tea.Msg {
		output, err := m.context.RunCommandImmediate(jj.OpLog(config.Current.OpLog.Limit))
		if err != nil {
			panic(err)
		}

		rows := parseRows(bytes.NewReader(output))
		return updateOpLogMsg{Rows: rows}
	}
}

func New(context *context.MainContext, width int, height int) *Model {
	keyMap := config.Current.GetKeyMap()
	w := graph.NewRenderer(width, height)
	return &Model{
		List:      list.NewList[*models.OperationLogItem](),
		Sizeable:  &common.Sizeable{Width: width, Height: height},
		context:   context,
		w:         w,
		keymap:    keyMap,
		textStyle: common.DefaultPalette.Get("oplog text"),
	}
}
