package oplog

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/graph"
)

type Model struct {
	context   *context.MainContext
	w         *graph.Renderer
	keymap    config.KeyMappings[key.Binding]
	width     int
	height    int
	textStyle lipgloss.Style
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
	m.context.OpLog.Load()
	return nil
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			return m, common.Close
		case key.Matches(msg, m.keymap.Up):
			m.context.OpLog.Prev()
		case key.Matches(msg, m.keymap.Down):
			m.context.OpLog.Next()
		case key.Matches(msg, m.keymap.Diff):
			return m, func() tea.Msg {
				output, _ := m.context.RunCommandImmediate(jj.OpShow(m.context.OpLog.Current().OperationId))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, m.keymap.OpLog.Restore):
			return m, tea.Batch(common.Close, m.context.RunCommand(jj.OpRestore(m.context.OpLog.Current().OperationId), common.Refresh))
		}
	}
	return m, nil
}

func (m *Model) View() string {
	if m.context.OpLog.Items == nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "loading")
	}

	m.w.Reset()
	m.w.SetSize(m.width, m.height)
	renderer := newIterator(m.context.OpLog, m.width)
	content := m.w.Render(renderer)
	content = lipgloss.PlaceHorizontal(m.width, lipgloss.Left, content)
	return m.textStyle.MaxWidth(m.width).Render(content)
}

func New(context *context.MainContext, width int, height int) *Model {
	keyMap := config.Current.GetKeyMap()
	w := graph.NewRenderer(width, height)
	return &Model{
		context:   context,
		w:         w,
		keymap:    keyMap,
		width:     width,
		height:    height,
		textStyle: common.DefaultPalette.Get("oplog text"),
	}
}
