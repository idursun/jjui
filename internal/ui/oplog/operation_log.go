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
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type updateOpLogMsg struct {
	Rows []row
}

type OpLogClickedMsg struct {
	Index int
}

type OpLogScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (o OpLogScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	return OpLogScrollMsg{Delta: delta, Horizontal: horizontal}
}

var _ common.ImmediateModel = (*Model)(nil)

type Model struct {
	context          *context.MainContext
	listRenderer     *render.ListRenderer
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

func (m *Model) Cursor() int {
	return m.cursor
}

func (m *Model) SetCursor(index int) {
	if index >= 0 && index < len(m.rows) {
		m.cursor = index
		m.ensureCursorView = true
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

func (m *Model) Scroll(delta int) tea.Cmd {
	m.ensureCursorView = false
	currentStart := m.listRenderer.GetScrollOffset()
	desiredStart := currentStart + delta
	m.listRenderer.SetScrollOffset(desiredStart)
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		return m.handleIntent(msg)
	case updateOpLogMsg:
		m.rows = msg.Rows
		return m.updateSelection()
	case OpLogClickedMsg:
		if msg.Index >= 0 && msg.Index < len(m.rows) {
			m.cursor = msg.Index
			m.ensureCursorView = true
			return m.updateSelection()
		}
	case OpLogScrollMsg:
		if msg.Horizontal {
			return nil
		}
		return m.Scroll(msg.Delta)
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

	// Calculate step (convert page scroll to item count)
	step := delta
	if page {
		firstRowIndex := m.listRenderer.GetFirstRowIndex()
		lastRowIndex := m.listRenderer.GetLastRowIndex()
		span := max(lastRowIndex-firstRowIndex-1, 1)
		if step < 0 {
			step = -span
		} else {
			step = span
		}
	}

	// Calculate new cursor position
	totalItems := len(m.rows)
	newCursor := m.cursor + step
	if newCursor < 0 {
		newCursor = 0
	} else if newCursor >= totalItems {
		newCursor = totalItems - 1
	}

	m.SetCursor(newCursor)
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

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if m.rows == nil {
		content := lipgloss.Place(box.R.Dx(), box.R.Dy(), lipgloss.Center, lipgloss.Center, "loading")
		dl.AddDraw(box.R, content, 0)
		return
	}

	measure := func(index int) int {
		return len(m.rows[index].Lines)
	}

	renderItem := func(dl *render.DisplayContext, index int, itemRect cellbuf.Rectangle) {
		row := m.rows[index]
		isSelected := index == m.cursor
		styleOverride := m.textStyle
		if isSelected {
			styleOverride = m.selectedStyle
		}

		y := itemRect.Min.Y
		for _, line := range row.Lines {
			var content bytes.Buffer
			for _, segment := range line.Segments {
				content.WriteString(segment.Style.Inherit(styleOverride).Render(segment.Text))
			}
			lineContent := lipgloss.PlaceHorizontal(itemRect.Dx(), 0, content.String(), lipgloss.WithWhitespaceBackground(styleOverride.GetBackground()))
			lineRect := cellbuf.Rect(itemRect.Min.X, y, itemRect.Dx(), 1)
			dl.AddDraw(lineRect, lineContent, 0)
			y++
		}
	}

	clickMsg := func(index int) render.ClickMessage {
		return OpLogClickedMsg{Index: index}
	}

	m.listRenderer.Render(
		dl,
		layout.Box{R: box.R},
		len(m.rows),
		m.cursor,
		m.ensureCursorView,
		measure,
		renderItem,
		clickMsg,
	)

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
		context:       context,
		keymap:        keyMap,
		rows:          nil,
		cursor:        0,
		textStyle:     common.DefaultPalette.Get("oplog text"),
		selectedStyle: common.DefaultPalette.Get("oplog selected"),
	}
	m.listRenderer = render.NewListRenderer(OpLogScrollMsg{})
	return m
}
