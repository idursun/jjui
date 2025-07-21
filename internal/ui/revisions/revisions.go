package revisions

import (
	"bytes"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/idursun/jjui/internal/ui/operations/duplicate"

	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/operations/describe"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/graph"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/abandon"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/operations/evolog"
	"github.com/idursun/jjui/internal/ui/operations/rebase"
	"github.com/idursun/jjui/internal/ui/operations/squash"
)

type Model struct {
	rows              []parser.Row
	tag               uint64
	revisionToSelect  string
	offScreenRows     []parser.Row
	streamer          *graph.GraphStreamer
	rowsChan          <-chan parser.RowBatch
	controlChan       chan parser.ControlMsg
	hasMore           bool
	op                operations.Operation
	cursor            int
	width             int
	height            int
	context           *appContext.MainContext
	keymap            config.KeyMappings[key.Binding]
	output            string
	err               error
	quickSearch       string
	selectedRevisions map[string]bool
	previousOpLogId   string
	isLoading         bool
	w                 *graph.Renderer
	textStyle         lipgloss.Style
}

type updateRevisionsMsg struct {
	rows             []parser.Row
	selectedRevision string
}

type startRowsStreamingMsg struct {
	selectedRevision string
	tag              uint64
}

type appendRowsBatchMsg struct {
	rows    []parser.Row
	hasMore bool
	tag     uint64
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
		if m.selectedRevisions[row.Commit.GetChangeId()] {
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
		m.op = operations.NewDefault()
		return m, m.updateSelection()
	case common.UpdateRevSetMsg:
		return m, common.Refresh
	case common.QuickSearchMsg:
		m.quickSearch = string(msg)
		m.cursor = m.search(0)
		m.op = operations.NewDefault()
		m.w.ResetViewRange()
		return m, nil
	case common.CommandCompletedMsg:
		m.output = msg.Output
		m.err = msg.Err
		return m, nil
	case common.AutoRefreshMsg:
		id, _ := m.context.RunCommandImmediate(jj.OpLogId(true))
		currentOperationId := string(id)
		log.Println("Previous operation ID:", m.previousOpLogId, "Current operation ID:", currentOperationId)
		if currentOperationId != m.previousOpLogId {
			m.previousOpLogId = currentOperationId
			return m, common.RefreshAndKeepSelections
		}
	case common.RefreshMsg:
		if !msg.KeepSelections {
			m.selectedRevisions = make(map[string]bool)
		}
		m.isLoading = true
		cmd, _ := m.updateOperation(msg)
		if config.Current.ExperimentalLogBatchingEnabled {
			m.tag += 1
			return m, tea.Batch(m.loadStreaming(m.context.CurrentRevset, msg.SelectedRevision, m.tag), cmd)
		} else {
			return m, tea.Batch(m.load(m.context.CurrentRevset, msg.SelectedRevision), cmd)
		}
	case updateRevisionsMsg:
		m.isLoading = false
		m.updateGraphRows(msg.rows, msg.selectedRevision)
		return m, tea.Batch(m.highlightChanges, m.updateSelection(), func() tea.Msg {
			return common.UpdateRevisionsSuccessMsg{}
		})
	case startRowsStreamingMsg:
		m.offScreenRows = nil
		m.revisionToSelect = msg.selectedRevision
		log.Println("Starting streaming revisions message received with tag:", msg.tag)
		return m, m.requestMoreRows(msg.tag)
	case appendRowsBatchMsg:
		if msg.tag != m.tag {
			return m, nil
		}
		m.offScreenRows = append(m.offScreenRows, msg.rows...)
		m.hasMore = msg.hasMore
		m.isLoading = m.hasMore && len(m.offScreenRows) > 0

		if m.hasMore {
			// keep requesting rows until we reach the initial load count or the current cursor position
			if len(m.offScreenRows) < m.cursor+1 || len(m.offScreenRows) < m.w.LastRowIndex()+1 {
				return m, m.requestMoreRows(msg.tag)
			}
		} else if m.streamer != nil {
			m.streamer.Close()
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

		if (m.cursor < 0 || m.cursor >= len(m.rows)) && len(m.rows) > 0 {
			m.cursor = 0
		}

		cmds := []tea.Cmd{m.highlightChanges, m.updateSelection()}
		if !m.hasMore {
			cmds = append(cmds, func() tea.Msg {
				return common.UpdateRevisionsSuccessMsg{}
			})
		}
		return m, tea.Batch(cmds...)
	}

	if cmd, ok := m.updateOperation(msg); ok {
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
				return m, m.requestMoreRows(m.tag)
			}
		case key.Matches(msg, m.keymap.JumpToParent):
			immediate, _ := m.context.RunCommandImmediate(jj.GetParent(m.SelectedRevisions()))
			parentIndex := m.selectRevision(string(immediate))
			if parentIndex != -1 {
				m.cursor = parentIndex
			}
		case key.Matches(msg, m.keymap.JumpToChildren):
			immediate, _ := m.context.RunCommandImmediate(jj.GetFirstChild(m.SelectedRevision()))
			index := m.selectRevision(string(immediate))
			if index != -1 {
				m.cursor = index
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
				changeId := m.rows[m.cursor].Commit.GetChangeId()
				m.selectedRevisions[changeId] = !m.selectedRevisions[changeId]
			case key.Matches(msg, m.keymap.Cancel):
				m.op = operations.NewDefault()
			case key.Matches(msg, m.keymap.QuickSearchCycle):
				m.cursor = m.search(m.cursor + 1)
				m.w.ResetViewRange()
				return m, nil
			case key.Matches(msg, m.keymap.Details.Mode):
				m.op, cmd = details.NewOperation(m.context, m.SelectedRevision())
			case key.Matches(msg, m.keymap.InlineDescribe.Mode):
				m.op, cmd = describe.NewOperation(m.context, m.SelectedRevision().GetChangeId(), m.width)
				return m, cmd
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
				m.op, cmd = evolog.NewOperation(m.context, m.SelectedRevision(), m.width, m.height)
			case key.Matches(msg, m.keymap.Diff):
				return m, func() tea.Msg {
					changeId := m.SelectedRevision().GetChangeId()
					output, _ := m.context.RunCommandImmediate(jj.Diff(changeId, ""))
					return common.ShowDiffMsg(output)
				}
			case key.Matches(msg, m.keymap.Refresh):
				cmd = common.Refresh
			case key.Matches(msg, m.keymap.Squash.Mode):
				selectedRevisions := m.SelectedRevisions()
				parent, _ := m.context.RunCommandImmediate(jj.GetParent(selectedRevisions))
				parentIdx := m.selectRevision(string(parent))
				if parentIdx != -1 {
					m.cursor = parentIdx
				} else if m.cursor < len(m.rows)-1 {
					m.cursor++
				}
				m.op = squash.NewOperation(m.context, selectedRevisions)
			case key.Matches(msg, m.keymap.Rebase.Mode):
				m.op = rebase.NewOperation(m.context, m.SelectedRevisions(), rebase.SourceRevision, rebase.TargetDestination)
			case key.Matches(msg, m.keymap.Duplicate.Mode):
				m.op = duplicate.NewOperation(m.context, m.SelectedRevisions(), duplicate.TargetDestination)
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
		return m.context.SetSelectedItem(appContext.SelectedRevision{ChangeId: selectedRevision.GetChangeId()})
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

func (m *Model) updateGraphRows(rows []parser.Row, selectedRevision string) {
	if rows == nil {
		rows = []parser.Row{}
	}

	currentSelectedRevision := selectedRevision
	if cur := m.SelectedRevision(); currentSelectedRevision == "" && cur != nil {
		currentSelectedRevision = cur.GetChangeId()
	}
	m.rows = rows

	if len(m.rows) > 0 {
		m.cursor = m.selectRevision(currentSelectedRevision)
		if m.cursor == -1 {
			m.cursor = m.selectRevision("@")
		}
		if m.cursor == -1 {
			m.cursor = 0
		}
	} else {
		m.cursor = 0
	}
}

func (m *Model) View() string {
	if len(m.rows) == 0 {
		if m.isLoading {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "loading")
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "(no matching revisions)")
	}

	renderer := graph.NewDefaultRowIterator(m.rows, m.width)
	renderer.Op = m.op
	renderer.Cursor = m.cursor
	renderer.Selections = m.selectedRevisions
	renderer.SearchText = m.quickSearch
	renderer.TextStyle = common.DefaultPalette.Get("revisions text")
	renderer.SelectedStyle = common.DefaultPalette.Get("revisions selected").Inline(true)
	m.w.SetSize(m.width, m.height)
	output := m.w.Render(renderer)
	output = m.textStyle.MaxWidth(m.width).Render(output)
	return lipgloss.Place(m.width, m.height, 0, 0, output)
}

func (m *Model) load(revset string, selectedRevision string) tea.Cmd {
	return func() tea.Msg {
		output, err := m.context.RunCommandImmediate(jj.Log(revset, config.Current.Limit))
		if err != nil {
			return common.UpdateRevisionsFailedMsg{
				Err:    err,
				Output: string(output),
			}
		}
		rows := parser.ParseRows(bytes.NewReader(output))
		return updateRevisionsMsg{rows, selectedRevision}
	}
}

func (m *Model) loadStreaming(revset string, selectedRevision string, tag uint64) tea.Cmd {
	if m.tag != tag {
		return nil
	}

	if m.streamer != nil {
		m.streamer.Close()
		m.streamer = nil
	}

	m.hasMore = false
	return func() tea.Msg {
		streamer, err := graph.NewGraphStreamer(m.context, revset)
		if err != nil {
			return common.UpdateRevisionsFailedMsg{
				Err:    err,
				Output: fmt.Sprintf("%v", err),
			}
		}
		m.streamer = streamer
		m.hasMore = true
		m.offScreenRows = nil
		log.Println("Starting streaming revisions with tag:", tag)
		return startRowsStreamingMsg{selectedRevision, tag}
	}
}

func (m *Model) requestMoreRows(tag uint64) tea.Cmd {
	return func() tea.Msg {
		if m.streamer == nil || !m.hasMore {
			return nil
		}
		batch := m.streamer.RequestMore()
		return appendRowsBatchMsg{batch.Rows, batch.HasMore, tag}
	}
}

func (m *Model) selectRevision(revision string) int {
	eqFold := func(other string) bool {
		return strings.EqualFold(other, revision)
	}

	idx := slices.IndexFunc(m.rows, func(row parser.Row) bool {
		if revision == "@" {
			return row.Commit.IsWorkingCopy
		}
		return eqFold(row.Commit.GetChangeId()) || eqFold(row.Commit.ChangeId) || eqFold(row.Commit.CommitId)
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

func New(c *appContext.MainContext) Model {
	keymap := config.Current.GetKeyMap()
	w := graph.NewRenderer(20, 10)
	return Model{
		context:           c,
		w:                 w,
		keymap:            keymap,
		rows:              nil,
		offScreenRows:     nil,
		op:                operations.NewDefault(),
		cursor:            0,
		width:             20,
		height:            10,
		selectedRevisions: make(map[string]bool),
		textStyle:         common.DefaultPalette.Get("revisions text"),
	}
}

func (m *Model) updateOperation(msg tea.Msg) (tea.Cmd, bool) {
	var cmd tea.Cmd
	if op, ok := m.op.(operations.OperationWithOverlay); ok {
		m.op, cmd = op.Update(msg)
		return cmd, true
	}
	return nil, false
}
