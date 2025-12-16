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
	_ list.IList         = (*Model)(nil)
	_ common.Model       = (*Model)(nil)
	_ common.IMouseAware = (*Model)(nil)
)

type Model struct {
	*common.ViewNode
	*common.MouseAware
	context          *context.MainContext
	renderer         *list.ListRenderer
	rows             []row
	cursor           int
	keymap           config.KeyMappings[key.Binding]
	textStyle        lipgloss.Style
	selectedStyle    lipgloss.Style
	ensureCursorView bool
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

func (m *Model) ClickAt(x, y int) tea.Cmd {
	if len(m.rows) == 0 {
		return nil
	}

	localY := y - m.Frame.Min.Y

	currentStart := m.renderer.ViewRange.Start
	if localY >= m.Height {
		localY = m.Height - 1
		if localY < 0 {
			return nil
		}
	}
	line := currentStart + localY
	row := m.rowAtLine(line)
	if row == -1 {
		return nil
	}

	m.cursor = row
	m.ensureCursorView = true
	return m.updateSelection()
}

func (m *Model) rowAtLine(line int) int {
	for _, rr := range m.renderer.RowRanges() {
		if line >= rr.StartLine && line < rr.EndLine {
			return rr.Row
		}
	}
	return -1
}

func (m *Model) Scroll(delta int) tea.Cmd {
	m.ensureCursorView = false
	desiredStart := m.renderer.ViewRange.Start + delta
	if desiredStart < 0 {
		desiredStart = 0
	}

	totalLines := m.renderer.AbsoluteLineCount()
	maxStart := totalLines - m.Height
	if maxStart < 0 {
		maxStart = 0
	}
	newStart := desiredStart
	if newStart > maxStart {
		newStart = maxStart
	}
	m.renderer.ViewRange.Start = newStart
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case updateOpLogMsg:
		m.rows = msg.Rows
		m.renderer.Reset()
	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			switch msg.Button {
			case tea.MouseButtonLeft:
				return m.ClickAt(msg.X, msg.Y)
			case tea.MouseButtonWheelUp:
				return m.Scroll(-3)
			case tea.MouseButtonWheelDown:
				return m.Scroll(3)
			}
			return nil
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Cancel):
			return tea.Batch(common.Close, common.Refresh, common.SelectionChanged)
		case key.Matches(msg, m.keymap.Up):
			if m.cursor > 0 {
				m.cursor--
				m.ensureCursorView = true
			}
		case key.Matches(msg, m.keymap.Down):
			if m.cursor < len(m.rows)-1 {
				m.cursor++
				m.ensureCursorView = true
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
	content := m.renderer.RenderWithOptions(list.RenderOptions{FocusIndex: m.cursor, EnsureFocusVisible: m.ensureCursorView})
	return m.textStyle.Render(content)
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

func New(context *context.MainContext) *Model {
	keyMap := config.Current.GetKeyMap()
	node := common.NewViewNode(0, 0)
	m := &Model{
		ViewNode:      node,
		MouseAware:    common.NewMouseAware(),
		context:       context,
		keymap:        keyMap,
		rows:          nil,
		cursor:        0,
		textStyle:     common.DefaultPalette.Get("oplog text"),
		selectedStyle: common.DefaultPalette.Get("oplog selected"),
	}
	m.renderer = list.NewRenderer(m, node)
	return m
}
