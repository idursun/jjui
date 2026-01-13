package oplog

import (
	"bytes"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type updateOpLogMsg struct {
	Rows []row
}

var (
	_ list.IList            = (*Model)(nil)
	_ list.IScrollableList  = (*Model)(nil)
	_ common.ImmediateModel = (*Model)(nil)
	_ common.IMouseAware    = (*Model)(nil)
)

type Model struct {
	*common.MouseAware
	context          *context.MainContext
	renderer         *list.ListRenderer
	rows             []row
	cursor           int
	keymap           config.KeyMappings[key.Binding]
	textStyle        lipgloss.Style
	selectedStyle    lipgloss.Style
	ensureCursorView bool
	frame            cellbuf.Rectangle
}

func (m *Model) Len() int {
	if m.rows == nil {
		return 0
	}
	return len(m.rows)
}

func (m *Model) Cursor() int {
	return m.cursor
}

func (m *Model) SetCursor(index int) {
	if index >= 0 && index < len(m.rows) {
		m.cursor = index
		m.ensureCursorView = true
	}
}

func (m *Model) VisibleRange() (int, int) {
	return m.renderer.FirstRowIndex, m.renderer.LastRowIndex
}

func (m *Model) ListName() string {
	return "operation log"
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
		m.keymap.ScrollUp,
		m.keymap.ScrollDown,
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

	localY := y - m.frame.Min.Y

	currentStart := m.renderer.ViewRange.Start
	if localY >= m.renderer.ViewRange.Height {
		localY = m.renderer.ViewRange.Height - 1
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
	maxStart := totalLines - m.renderer.ViewRange.Height
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
	case intents.Intent:
		return m.handleIntent(msg)
	case updateOpLogMsg:
		m.rows = msg.Rows
		m.renderer.Reset()
		return m.updateSelection()
	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			switch msg.Button {
			case tea.MouseButtonLeft:
				return m.ClickAt(msg.X, msg.Y)
			case tea.MouseButtonWheelUp:
				return intents.Invoke(intents.OpLogNavigate{Delta: -3, IsPage: false})
			case tea.MouseButtonWheelDown:
				return intents.Invoke(intents.OpLogNavigate{Delta: 3, IsPage: false})
			}
			return nil
		}
	case tea.KeyMsg:
		return m.keyToIntent(msg)
	}
	return nil
}

func (m *Model) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent := intent.(type) {
	case intents.OpLogNavigate:
		return m.navigate(intent.Delta, intent.IsPage)
	case intents.OpLogClose:
		return m.close()
	case intents.OpLogShowDiff:
		return m.showDiff(intent)
	case intents.OpLogRestore:
		return m.restore(intent)
	case intents.OpLogRevert:
		return m.revert(intent)
	}
	return nil
}

func (m *Model) keyToIntent(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keymap.Cancel):
		return intents.Invoke(intents.OpLogClose{})
	case key.Matches(msg, m.keymap.Up, m.keymap.ScrollUp):
		return intents.Invoke(intents.OpLogNavigate{
			Delta:  -1,
			IsPage: key.Matches(msg, m.keymap.ScrollUp),
		})
	case key.Matches(msg, m.keymap.Down, m.keymap.ScrollDown):
		return intents.Invoke(intents.OpLogNavigate{
			Delta:  1,
			IsPage: key.Matches(msg, m.keymap.ScrollDown),
		})
	case key.Matches(msg, m.keymap.Diff):
		return intents.Invoke(intents.OpLogShowDiff{})
	case key.Matches(msg, m.keymap.OpLog.Restore):
		return intents.Invoke(intents.OpLogRestore{})
	case key.Matches(msg, m.keymap.OpLog.Revert):
		return intents.Invoke(intents.OpLogRevert{})
	}
	return nil
}

func (m *Model) navigate(delta int, page bool) tea.Cmd {
	if len(m.rows) == 0 {
		return nil
	}

	result := list.Scroll(m, delta, page)

	if result.NavigateMessage != nil {
		return func() tea.Msg { return *result.NavigateMessage }
	}

	m.SetCursor(result.NewCursor)
	return m.updateSelection()
}

func (m *Model) updateSelection() tea.Cmd {
	if len(m.rows) == 0 {
		return nil
	}
	return m.context.SetSelectedItem(context.SelectedOperation{OperationId: m.rows[m.cursor].OperationId})
}

func (m *Model) close() tea.Cmd {
	return tea.Batch(common.Close, common.Refresh, common.SelectionChanged(m.context.SelectedItem))
}

func (m *Model) showDiff(intent intents.OpLogShowDiff) tea.Cmd {
	opId := intent.OperationId
	if opId == "" {
		if len(m.rows) == 0 {
			return nil
		}
		opId = m.rows[m.cursor].OperationId
	}
	return func() tea.Msg {
		output, _ := m.context.RunCommandImmediate(jj.OpShow(opId))
		return common.ShowDiffMsg(output)
	}
}

func (m *Model) restore(intent intents.OpLogRestore) tea.Cmd {
	opId := intent.OperationId
	if opId == "" {
		if len(m.rows) == 0 {
			return nil
		}
		opId = m.rows[m.cursor].OperationId
	}
	return tea.Batch(common.Close, m.context.RunCommand(jj.OpRestore(opId), common.Refresh))
}

func (m *Model) revert(intent intents.OpLogRevert) tea.Cmd {
	opId := intent.OperationId
	if opId == "" {
		if len(m.rows) == 0 {
			return nil
		}
		opId = m.rows[m.cursor].OperationId
	}
	return tea.Batch(common.Close, m.context.RunCommand(jj.OpRevert(opId), common.Refresh))
}

func (m *Model) ViewRect(dl *render.DisplayList, box layout.Box) {
	m.frame = box.R
	m.renderer.ViewRange.Width = box.R.Dx()
	m.renderer.ViewRange.Height = box.R.Dy()
	if m.rows == nil {
		content := lipgloss.Place(m.renderer.ViewRange.Width, m.renderer.ViewRange.Height, lipgloss.Center, lipgloss.Center, "loading")
		dl.AddDraw(box.R, content, 0)
		return
	}

	m.renderer.Reset()
	content := m.renderer.RenderWithOptions(list.RenderOptions{FocusIndex: m.cursor, EnsureFocusVisible: m.ensureCursorView})
	dl.AddDraw(box.R, m.textStyle.Render(content), 0)
	m.ensureCursorView = false
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
	m := &Model{
		MouseAware:    common.NewMouseAware(),
		context:       context,
		keymap:        keyMap,
		rows:          nil,
		cursor:        0,
		textStyle:     common.DefaultPalette.Get("oplog text"),
		selectedStyle: common.DefaultPalette.Get("oplog selected"),
	}
	m.renderer = list.NewRenderer(m, 0, 0)
	return m
}

func (m *Model) Frame() cellbuf.Rectangle {
	return m.frame
}
