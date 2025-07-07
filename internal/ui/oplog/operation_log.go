package oplog

import (
	"bytes"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/graph"
)

type updateOpLogMsg struct {
	Rows []row
}

type Model struct {
	context *context.MainContext
	w       *graph.Renderer
	rows    []row
	cursor  int
	keymap  config.KeyMappings[key.Binding]
	width   int
	height  int
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{m.keymap.Up, m.keymap.Down, m.keymap.Cancel, m.keymap.Diff, m.keymap.OpLog.Restore}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Width() int {
	return m.width
}

func (m *Model) Height() int {
	return m.height
}

func (m *Model) SetWidth(w int) {
	m.width = w
}

func (m *Model) SetHeight(h int) {
	m.height = h
}

func (m *Model) Init() tea.Cmd {
	return m.load()
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateOpLogMsg:
		m.rows = msg.Rows
		m.w.Reset()
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			m.cursor = m.w.FindRowIndexAtLine(msg.Y - 1)
			return m, m.updateSelection()
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			return m, common.Close
		case key.Matches(msg, m.keymap.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keymap.Down):
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keymap.Diff):
			return m, func() tea.Msg {
				output, _ := m.context.RunCommandImmediate(jj.OpShow(m.rows[m.cursor].OperationId))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, m.keymap.OpLog.Restore):
			return m, tea.Batch(common.Close, m.context.RunCommand(jj.OpRestore(m.rows[m.cursor].OperationId), common.Refresh))
		}
	}
	return m, m.updateSelection()
}

func (m *Model) updateSelection() tea.Cmd {
	if m.rows == nil {
		return nil
	}
	return m.context.SetSelectedItem(context.SelectedOperation{OperationId: m.rows[m.cursor].OperationId})
}

func (m *Model) View() string {
	if m.rows == nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "loading")
	}

	m.w.Reset()
	m.w.SetSize(m.width, m.height)
	renderer := newIterator(m.rows, m.cursor, m.width)
	content := m.w.Render(renderer)
	content = lipgloss.PlaceHorizontal(m.width, lipgloss.Left, content)
	return common.DefaultPalette.Normal.MaxWidth(m.width).Render(content)
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
		context: context,
		w:       w,
		keymap:  keyMap,
		rows:    nil,
		cursor:  0,
		width:   width,
		height:  height,
	}
}
