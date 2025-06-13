package revisions

import (
	"bufio"
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

const defaultBatchSize = 50

type viewRange struct {
	start        int
	end          int
	lastRowIndex int
}

func (v *viewRange) reset() {
	v.start = 0
	v.end = 0
	v.lastRowIndex = -1
}

type Model struct {
	rows             []graph.Row
	revisionToSelect string
	offScreenRows    []graph.Row
	rowsChan         <-chan graph.RowBatch
	controlChan      chan graph.ControlMsg
	hasMore          bool
	op               operations.Operation
	revsetValue      string
	viewRange        *viewRange
	cursor           int
	width            int
	height           int
	context          context.AppContext
	keymap           config.KeyMappings[key.Binding]
	output           string
	err              error
	quickSearch      string
}

type updateRevisionsMsg struct {
	rows             []graph.Row
	selectedRevision string
}

type startRowsStreamingMsg struct {
	rowsChan         <-chan graph.RowBatch
	selectedRevision string
}

type appendRowsBatchMsg struct {
	rows    []graph.Row
	hasMore bool
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
	if m.cursor >= len(m.rows) || m.cursor < 0 {
		return nil
	}
	return m.rows[m.cursor].Commit
}

func (m *Model) SelectedRevisions() jj.SelectedRevisions {
	var selected []*jj.Commit
	for _, row := range m.rows {
		if row.IsSelected {
			selected = append(selected, row.Commit)
		}
	}
	if len(selected) == 0 {
		return jj.NewSelectedRevisions(m.SelectedRevision())
	}
	return jj.NewSelectedRevisions(selected...)
}

func (m *Model) Init() tea.Cmd {
	return common.RefreshAndSelect("@")
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
		m.quickSearch = string(msg)
		m.cursor = m.search(0)
		m.op = operations.NewDefault(m.context)
		m.viewRange.reset()
		return m, nil
	case common.CommandCompletedMsg:
		m.output = msg.Output
		m.err = msg.Err
		return m, nil
	case common.RefreshMsg:
		if config.Current.ExperimentalLogBatchingEnabled {
			return m, m.loadStreaming(m.revsetValue, msg.SelectedRevision)
		} else {
			return m, m.load(m.revsetValue, msg.SelectedRevision)
		}
	case updateRevisionsMsg:
		m.updateGraphRows(msg.rows, msg.selectedRevision)
		return m, tea.Batch(m.highlightChanges, m.updateSelection())
	case startRowsStreamingMsg:
		m.offScreenRows = nil
		m.revisionToSelect = msg.selectedRevision
		m.rowsChan = msg.rowsChan
		return m, m.requestMoreRows(m.rowsChan)
	case appendRowsBatchMsg:
		m.offScreenRows = append(m.offScreenRows, msg.rows...)
		m.hasMore = msg.hasMore

		if m.hasMore {
			// keep requesting rows until we reach the initial load count or the current cursor position
			if len(m.offScreenRows) < m.cursor+1 || len(m.offScreenRows) < m.viewRange.lastRowIndex+1 {
				return m, m.requestMoreRows(m.rowsChan)
			}
		} else if m.controlChan != nil {
			close(m.controlChan)
			m.rowsChan = nil
		}

		currentSelectedRevision := m.SelectedRevision()
		m.rows = m.offScreenRows
		if m.revisionToSelect != "" {
			m.cursor = m.selectRevision(m.revisionToSelect)
			m.revisionToSelect = ""
		}

		if m.cursor == -1 && currentSelectedRevision != nil {
			m.cursor = m.selectRevision(currentSelectedRevision.GetChangeId())
		}

		if m.cursor == -1 && len(m.rows) > 0 {
			m.cursor = 0
		}

		return m, tea.Batch(m.highlightChanges, m.updateSelection())
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
			} else if m.hasMore {
				return m, m.requestMoreRows(m.rowsChan)
			}
		case key.Matches(msg, m.keymap.JumpToParent):
			immediate, _ := m.context.RunCommandImmediate(jj.GetParent(m.SelectedRevision().GetChangeId()))
			parentIndex := m.selectRevision(string(immediate))
			if parentIndex != -1 {
				m.cursor = parentIndex
			}
		case key.Matches(msg, m.keymap.JumpToWorkingCopy):
			workingCopyIndex := m.selectRevision("@")
			if workingCopyIndex != -1 {
				m.cursor = workingCopyIndex
			}
			return m, m.updateSelection()
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
			case key.Matches(msg, m.keymap.QuickSearchCycle):
				m.cursor = m.search(m.cursor + 1)
				m.viewRange.reset()
				return m, nil
			case key.Matches(msg, m.keymap.Details.Mode):
				m.op, cmd = details.NewOperation(m.context, m.SelectedRevision())
			case key.Matches(msg, m.keymap.New):
				cmd = m.context.RunCommand(jj.New(m.SelectedRevisions()), common.RefreshAndSelect("@"))
			case key.Matches(msg, m.keymap.Commit):
				cmd = m.context.RunInteractiveCommand(jj.CommitWorkingCopy(), common.Refresh)
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
				m.op = abandon.NewOperation(m.context, selections)
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
				parent, _ := m.context.RunCommandImmediate(jj.GetParent(m.SelectedRevision().GetChangeId()))
				m.op = squash.NewOperation(m.context, m.SelectedRevisions())
				parentIdx := m.selectRevision(string(parent))
				if parentIdx != -1 {
					m.cursor = parentIdx
				} else if m.cursor < len(m.rows)-1 {
					m.cursor++
				}
			case key.Matches(msg, m.keymap.Rebase.Mode):
				m.op = rebase.NewOperation(m.context, m.SelectedRevisions(), rebase.SourceRevision, rebase.TargetDestination)
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
	m.viewRange.reset()
}

func (m *Model) View() string {
	if len(m.rows) == 0 {
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
	lastRenderedRowIndex := -1
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
		nodeRenderer := graph.DefaultRowDecorator{
			Palette:             common.DefaultPalette,
			Op:                  m.op,
			HighlightBackground: highlightBackground,
			SearchText:          m.quickSearch,
			IsHighlighted:       i == m.cursor,
			IsSelected:          row.IsSelected,
			Width:               m.width,
		}

		graph.RenderRow(&w, row, nodeRenderer)
		if i == m.cursor {
			selectedLineEnd = w.LineCount()
		}
		if selectedLineEnd > 0 && w.LineCount() > h && w.LineCount() > m.viewRange.end {
			lastRenderedRowIndex = i
			break
		}
	}
	if lastRenderedRowIndex == -1 {
		lastRenderedRowIndex = len(m.rows) - 1
	}

	m.viewRange.lastRowIndex = lastRenderedRowIndex
	if selectedLineStart <= m.viewRange.start {
		m.viewRange.start = selectedLineStart
		m.viewRange.end = selectedLineStart + h
	} else if selectedLineEnd > m.viewRange.end {
		m.viewRange.end = selectedLineEnd
		m.viewRange.start = selectedLineEnd - h
	}

	content := w.String(m.viewRange.start, m.viewRange.end)
	content = lipgloss.PlaceHorizontal(m.Width(), lipgloss.Left, content)

	return common.DefaultPalette.Normal.MaxWidth(m.width).Render(content)
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

func (m *Model) loadStreaming(revset string, selectedRevision string) tea.Cmd {
	if m.hasMore {
		m.controlChan <- graph.Close
		close(m.controlChan)
		m.controlChan = nil
		m.rowsChan = nil
	}
	m.hasMore = false
	return func() tea.Msg {
		output, err := m.context.RunCommandStreaming(jj.Log(revset))
		if err != nil {
			return common.UpdateRevisionsFailedMsg{
				Err: err,
				// TODO: change this to the actual error output
				Output: "failed",
			}
		}
		m.controlChan = make(chan graph.ControlMsg, 1)
		rowsChan, _ := graph.ParseRowsStreaming(bufio.NewReader(output), m.controlChan, defaultBatchSize)
		return startRowsStreamingMsg{rowsChan, selectedRevision}
	}
}

func (m *Model) requestMoreRows(rowsChan <-chan graph.RowBatch) tea.Cmd {
	return func() tea.Msg {
		m.controlChan <- graph.RequestMore
		batch := <-rowsChan
		return appendRowsBatchMsg{batch.Rows, batch.HasMore}
	}
}

func (m *Model) selectRevision(revision string) int {
	idx := slices.IndexFunc(m.rows, func(row graph.Row) bool {
		if revision == "@" {
			return row.Commit.IsWorkingCopy
		}
		return row.Commit.GetChangeId() == revision || row.Commit.ChangeId == revision || row.Commit.CommitId == revision
	})
	return idx
}

func (m *Model) search(startIndex int) int {
	if m.quickSearch == "" {
		return m.cursor
	}

	n := len(m.rows)
	for i := startIndex; i < n+startIndex; i++ {
		c := i % n
		row := &m.rows[c]
		for _, line := range row.Lines {
			for _, segment := range line.Segments {
				if segment.Text != "" && strings.Contains(segment.Text, m.quickSearch) {
					return c
				}
			}
		}
	}
	return m.cursor
}

func (m *Model) CurrentOperation() operations.Operation {
	return m.op
}

func (m *Model) GetCommitIds() []string {
	var commitIds []string
	for _, row := range m.rows {
		commitIds = append(commitIds, row.Commit.CommitId)
	}
	return commitIds
}

func New(c context.AppContext, revset string) Model {
	v := viewRange{start: 0, end: 0, lastRowIndex: -1}
	keymap := c.KeyMap()
	return Model{
		context:       c,
		keymap:        keymap,
		revsetValue:   revset,
		rows:          nil,
		offScreenRows: nil,
		viewRange:     &v,
		op:            operations.NewDefault(c),
		cursor:        0,
		width:         20,
		height:        10,
	}
}
