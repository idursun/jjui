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
	"github.com/idursun/jjui/internal/ui/context"
)

type updateOpLogMsg struct {
	Rows []row
}

var (
	_ list.IList   = (*Model)(nil)
	_ common.Model = (*Model)(nil)
)

type Model struct {
	*common.Sizeable
	context       *context.MainContext
	renderer      *list.ListRenderer
	rows          []row
	cursor        int
	keymap        config.KeyMappings[key.Binding]
	textStyle     lipgloss.Style
	selectedStyle lipgloss.Style
}

func (m *Model) Len() int {
	if m.rows == nil {
		return 0
	}
	return len(m.rows)
}

func (m *Model) GetItemRenderer(index int) list.IItemRenderer {
	item := m.rows[index]
	style := m.textStyle
	if index == m.cursor {
		style = m.selectedStyle
	}
	return &itemRenderer{
		row:   item,
		style: style,
	}
}

func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keymap.Up,
		m.keymap.Down,
		m.keymap.Cancel,
		m.keymap.Diff,
		m.keymap.OpLog.Restore,
		m.keymap.OpLog.Revert,
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) Init() tea.Cmd {
	return m.load()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case updateOpLogMsg:
		m.rows = msg.Rows
		m.renderer.Reset()
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			return common.Close
		case key.Matches(msg, m.keymap.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keymap.Down):
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keymap.Diff):
			return func() tea.Msg {
				output, _ := m.context.RunCommandImmediate(jj.OpShow(m.rows[m.cursor].OperationId))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, m.keymap.OpLog.Restore):
			return tea.Batch(common.Close, m.context.RunCommand(jj.OpRestore(m.rows[m.cursor].OperationId), common.Refresh))
		case key.Matches(msg, m.keymap.OpLog.Revert):
			return tea.Batch(common.Close, m.context.RunCommand(jj.OpRevert(m.rows[m.cursor].OperationId), common.Refresh))
		}

	}
	return m.updateSelection()
}

func (m *Model) updateSelection() tea.Cmd {
	if m.rows == nil {
		return nil
	}
	return m.context.SetSelectedItem(context.SelectedOperation{OperationId: m.rows[m.cursor].OperationId})
}

func (m *Model) View() string {
	if m.rows == nil {
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, "loading")
	}

	m.renderer.Reset()
	m.renderer.SetWidth(m.Width)
	m.renderer.SetHeight(m.Height)
	content := m.renderer.Render(m.cursor)
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
	m := &Model{
		Sizeable:      &common.Sizeable{Width: width, Height: height},
		context:       context,
		keymap:        keyMap,
		rows:          nil,
		cursor:        0,
		textStyle:     common.DefaultPalette.Get("oplog text"),
		selectedStyle: common.DefaultPalette.Get("oplog selected"),
	}
	m.renderer = list.NewRenderer(m, common.NewSizeable(width, height))
	return m
}
