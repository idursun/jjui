package revisions

import (
	"bytes"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/graph"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/abandon"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/operations/evolog"
	"github.com/idursun/jjui/internal/ui/operations/rebase"
	"github.com/idursun/jjui/internal/ui/operations/squash"
	"github.com/idursun/jjui/internal/ui/revset"
)

type viewRange struct {
	start int
	end   int
}

var normalStyle = lipgloss.NewStyle()

type Model struct {
	rows        []graph.Row
	op          operations.Operation
	revsetValue string
	viewRange   *viewRange
	cursor      int
	width       int
	height      int
	context     context.AppContext
	keymap      config.KeyMappings[key.Binding]
	output      string
	err         error
}

type updateRevisionsMsg struct {
	rows             []graph.Row
	selectedRevision string
}

func (m *Model) IsFocused() bool {
	if _, ok := m.op.(common.Focusable); ok {
		return true
	}
	return false
}

func (m *Model) InNormalMode() bool {
	if _, ok := m.op.(*operations.Default); ok {
		return true
	}
	return false
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

func (m *Model) ShortHelp() []key.Binding {
	if op, ok := m.op.(help.KeyMap); ok {
		return op.ShortHelp()
	}
	return (&operations.Default{}).ShortHelp()
}

func (m *Model) FullHelp() [][]key.Binding {
	if op, ok := m.op.(help.KeyMap); ok {
		return op.FullHelp()
	}
	return [][]key.Binding{m.ShortHelp()}
}

func (m *Model) SelectedRevision() *jj.Commit {
	if m.cursor >= len(m.rows) {
		return nil
	}
	return m.rows[m.cursor].Commit
}

func (m *Model) SelectedRevisions() []*jj.Commit {
	var selected []*jj.Commit
	for _, row := range m.rows {
		if row.IsSelected {
			selected = append(selected, row.Commit)
		}
	}
	if len(selected) == 0 {
		return []*jj.Commit{m.SelectedRevision()}
	}
	return selected
}

func (m *Model) Init() tea.Cmd {
	return common.Refresh
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.CloseViewMsg:
		m.op = operations.NewDefault(m.context)
		return m, m.updateSelection()
	case revset.UpdateRevSetMsg:
		m.revsetValue = string(msg)
		return m, common.Refresh
	case common.QuickSearchMsg:
		m.cursor = m.search(string(msg))
		m.op = operations.NewDefault(m.context)
		m.viewRange = &viewRange{start: 0, end: 0}
		return m, nil
	case common.CommandCompletedMsg:
		m.output = msg.Output
		m.err = msg.Err
		return m, nil
	case common.RefreshMsg:
		return m, m.load(m.revsetValue, msg.SelectedRevision)
	case updateRevisionsMsg:
		m.updateGraphRows(msg.rows, msg.selectedRevision)
		return m, m.highlightChanges
	}

	if op, ok := m.op.(operations.OperationWithOverlay); ok {
		var cmd tea.Cmd
		m.op, cmd = op.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keymap.Down):
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		default:
			if op, ok := m.op.(operations.HandleKey); ok {
				cmd = op.HandleKey(msg)
				break
			}

			switch {
			case key.Matches(msg, m.keymap.ToggleSelect):
				m.rows[m.cursor].IsSelected = !m.rows[m.cursor].IsSelected
			case key.Matches(msg, m.keymap.Cancel):
				m.op = operations.NewDefault(m.context)
			case key.Matches(msg, m.keymap.Details.Mode):
				m.op, cmd = details.NewOperation(m.context, m.SelectedRevision())
			case key.Matches(msg, m.keymap.New):
				selections := m.SelectedRevisions()
				var changeIds []string
				for _, s := range selections {
					changeIds = append(changeIds, s.GetChangeId())
				}
				cmd = m.context.RunCommand(jj.New(changeIds...), common.RefreshAndSelect("@"))
			case key.Matches(msg, m.keymap.Edit):
				cmd = m.context.RunCommand(jj.Edit(m.SelectedRevision().GetChangeId()), common.Refresh)
			case key.Matches(msg, m.keymap.Diffedit):
				changeId := m.SelectedRevision().GetChangeId()
				cmd = m.context.RunInteractiveCommand(jj.DiffEdit(changeId), common.Refresh)
			case key.Matches(msg, m.keymap.Absorb):
				changeId := m.SelectedRevision().GetChangeId()
				cmd = m.context.RunCommand(jj.Absorb(changeId), common.Refresh)
			case key.Matches(msg, m.keymap.Abandon):
				selections := m.SelectedRevisions()
				var changeIds []string
				for _, s := range selections {
					changeIds = append(changeIds, s.GetChangeId())
				}
				m.op = abandon.NewOperation(m.context, changeIds)
			case key.Matches(msg, m.keymap.Bookmark.Set):
				m.op, cmd = bookmark.NewSetBookmarkOperation(m.context, m.SelectedRevision().GetChangeId())
			case key.Matches(msg, m.keymap.Split):
				currentRevision := m.SelectedRevision().GetChangeId()
				return m, m.context.RunInteractiveCommand(jj.Split(currentRevision, []string{}), common.Refresh)
			case key.Matches(msg, m.keymap.Describe):
				currentRevision := m.SelectedRevision().GetChangeId()
				return m, m.context.RunInteractiveCommand(jj.Describe(currentRevision), common.Refresh)
			case key.Matches(msg, m.keymap.Evolog):
				m.op, cmd = evolog.NewOperation(m.context, m.SelectedRevision().GetChangeId(), m.width, m.height)
			case key.Matches(msg, m.keymap.Diff):
				return m, func() tea.Msg {
					changeId := m.SelectedRevision().GetChangeId()
					output, _ := m.context.RunCommandImmediate(jj.Diff(changeId, ""))
					return common.ShowDiffMsg(output)
				}
			case key.Matches(msg, m.keymap.Refresh):
				cmd = common.Refresh
			case key.Matches(msg, m.keymap.Squash):
				m.op = squash.NewOperation(m.context, m.SelectedRevision().ChangeId)
				if m.cursor < len(m.rows)-1 {
					m.cursor++
				}
			case key.Matches(msg, m.keymap.Rebase.Mode):
				m.op = rebase.NewOperation(m.context, m.SelectedRevision().ChangeId, rebase.SourceRevision, rebase.TargetDestination)
			case key.Matches(msg, m.keymap.Quit):
				return m, tea.Quit
			}
		}
	}

	if curSelected := m.SelectedRevision(); curSelected != nil {
		if op, ok := m.op.(operations.TracksSelectedRevision); ok {
			op.SetSelectedRevision(curSelected)
		}
		return m, tea.Batch(m.updateSelection(), cmd)
	}
	return m, cmd
}

func (m *Model) updateSelection() tea.Cmd {
	if selectedRevision := m.SelectedRevision(); selectedRevision != nil {
		return m.context.SetSelectedItem(context.SelectedRevision{ChangeId: selectedRevision.GetChangeId()})
	}
	return nil
}

func (m *Model) highlightChanges() tea.Msg {
	if m.err != nil || m.output == "" {
		return nil
	}

	changes := strings.Split(m.output, "\n")
	for _, change := range changes {
		if !strings.HasPrefix(change, " ") {
			continue
		}
		line := strings.Trim(change, "\n ")
		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")
		if len(parts) > 0 {
			for i := range m.rows {
				row := &m.rows[i]
				if row.Commit.GetChangeId() == parts[0] {
					row.IsAffected = true
					break
				}
			}
		}
	}
	return nil
}

func (m *Model) updateGraphRows(rows []graph.Row, selectedRevision string) {
	if rows == nil {
		return
	}

	currentSelectedRevision := selectedRevision
	if cur := m.SelectedRevision(); currentSelectedRevision == "" && cur != nil {
		currentSelectedRevision = cur.GetChangeId()
	}
	m.rows = rows
	m.cursor = m.selectRevision(currentSelectedRevision)
	if m.cursor == -1 {
		m.cursor = m.selectRevision("@")
	}
	if m.cursor == -1 {
		m.cursor = 0
	}
	m.viewRange.start = 0
	m.viewRange.end = 0
}

func (m *Model) View() string {
	if m.rows == nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "loading")
	}

	h := m.height
	viewHeight := m.viewRange.end - m.viewRange.start
	if viewHeight != h {
		m.viewRange.end = m.viewRange.start + h
	}

	highlightBackground := lipgloss.AdaptiveColor{
		Light: config.Current.UI.HighlightLight,
		Dark:  config.Current.UI.HighlightDark,
	}

	var w graph.Renderer
	selectedLineStart := -1
	selectedLineEnd := -1
	for i, row := range m.rows {
		if i == m.cursor {
			selectedLineStart = w.LineCount()
		} else {
			rowLineCount := len(row.Lines)
			if rowLineCount+w.LineCount() < m.viewRange.start {
				w.SkipLines(rowLineCount)
				continue
			}
		}
		nodeRenderer := &graph.DefaultRowDecorator{
			Palette:             common.DefaultPalette,
			Op:                  m.op,
			HighlightBackground: highlightBackground,
			IsHighlighted:       i == m.cursor,
			IsSelected:          row.IsSelected,
		}

		graph.RenderRow(&w, row, nodeRenderer, nodeRenderer.IsHighlighted, m.width)
		if i == m.cursor {
			selectedLineEnd = w.LineCount()
		}
		if selectedLineEnd > 0 && w.LineCount() > h && w.LineCount() > m.viewRange.end {
			break
		}
	}

	if selectedLineStart <= m.viewRange.start {
		m.viewRange.start = selectedLineStart
		m.viewRange.end = selectedLineStart + h
	} else if selectedLineEnd > m.viewRange.end {
		m.viewRange.end = selectedLineEnd
		m.viewRange.start = selectedLineEnd - h
	}

	content := w.String(m.viewRange.start, m.viewRange.end)
	content = lipgloss.PlaceHorizontal(m.Width(), lipgloss.Left, content)

	return normalStyle.MaxWidth(m.width).Render(content)
}

func (m *Model) load(revset string, selectedRevision string) tea.Cmd {
	return func() tea.Msg {
		output, err := m.context.RunCommandImmediate(jj.Log(revset))
		if err != nil {
			return common.UpdateRevisionsFailedMsg{
				Err:    err,
				Output: string(output),
			}
		}
		rows := graph.ParseRows(bytes.NewReader(output))
		return updateRevisionsMsg{rows, selectedRevision}
	}
}

func (m *Model) selectRevision(revision string) int {
	idx := slices.IndexFunc(m.rows, func(row graph.Row) bool {
		if revision == "@" {
			return row.Commit.IsWorkingCopy
		}
		return row.Commit.GetChangeId() == revision || row.Commit.ChangeId == revision
	})
	return idx
}

func (m *Model) search(query string) int {
	if query == "" {
		return m.cursor
	}

	idx := slices.IndexFunc(m.rows, func(row graph.Row) bool {
		for _, line := range row.Lines {
			for _, segment := range line.Segments {
				if segment.Text != "" && strings.Contains(segment.Text, query) {
					return true
				}
			}
		}
		return false
	})
	if idx != -1 {
		return idx
	}
	return m.cursor
}

func (m *Model) CurrentOperation() operations.Operation {
	return m.op
}

func New(c context.AppContext, revset string) Model {
	v := viewRange{start: 0, end: 0}
	keymap := c.KeyMap()
	return Model{context: c,
		keymap:      keymap,
		revsetValue: revset,
		rows:        nil,
		viewRange:   &v,
		op:          operations.NewDefault(c),
		cursor:      0,
		width:       20,
		height:      10,
	}
}
