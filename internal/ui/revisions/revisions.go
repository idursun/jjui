package revisions

import (
	"bytes"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/operations/ace_jump"
	"github.com/idursun/jjui/internal/ui/operations/duplicate"
	"github.com/idursun/jjui/internal/ui/operations/revert"
	"github.com/idursun/jjui/internal/ui/operations/set_parents"
	"github.com/idursun/jjui/internal/ui/view"

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

const (
	scopeDetails        view.Scope = "details"
	scopeSquash         view.Scope = "squash"
	scopeRebase         view.Scope = "rebase"
	scopeInlineDescribe view.Scope = "inline_describe"
	scopeEvolog         view.Scope = "evolog"
	scopeRevert         view.Scope = "revert"
	scopeSetParents     view.Scope = "set_parents"
	scopeDuplicate      view.Scope = "duplicate"
	scopeAbandon        view.Scope = "abandon"
	scopeAceJump        view.Scope = "ace_jump"
	scopeSetBookmark    view.Scope = "set_bookmark"
)

var _ list.IList = (*Model)(nil)
var _ list.IListCursor = (*Model)(nil)
var _ common.ContextProvider = (*Model)(nil)
var _ view.IStatus = (*Model)(nil)
var _ view.IHasActionMap = (*Model)(nil)

type Model struct {
	*common.Sizeable
	rows             []parser.Row
	tag              atomic.Uint64
	revisionToSelect string
	offScreenRows    []parser.Row
	streamer         *graph.GraphStreamer
	hasMore          bool
	cursor           int
	context          *appContext.MainContext
	keymap           config.KeyMappings[key.Binding]
	output           string
	err              error
	quickSearch      string
	isLoading        bool
	renderer         *revisionListRenderer
	textStyle        lipgloss.Style
	dimmedStyle      lipgloss.Style
	selectedStyle    lipgloss.Style
	router           view.Router
	checkedRevisions map[string]bool
}

func (m *Model) Name() string {
	if len(m.router.Views) > 0 {
		if status, ok := m.router.Views[m.router.Scope].(view.IStatus); ok {
			return status.Name()
		}
	}
	return "revisions"
}

func (m *Model) GetActionMap() map[string]actions.Action {
	if len(m.router.Views) > 0 {
		if op, ok := m.router.Views[m.router.Scope].(view.IHasActionMap); ok {
			return op.GetActionMap()
		}
	}

	return config.Current.GetBindings("revisions")
}

func (m *Model) Read(value string) string {
	switch value {
	case jj.CheckedCommitIdsPlaceholder:
		if selectedRevisions := m.SelectedRevisions(); len(selectedRevisions.Revisions) > 0 {
			var commitIds []string
			for _, rev := range selectedRevisions.Revisions {
				commitIds = append(commitIds, rev.CommitId)
			}
			return strings.Join(commitIds, " | ")
		}
	case jj.ChangeIdPlaceholder:
		if current := m.SelectedRevision(); current != nil {
			return current.GetChangeId()
		}
	case jj.CommitIdPlaceholder:
		if current := m.SelectedRevision(); current != nil {
			return current.CommitId
		}
	}
	return m.router.Read(value)
}

func (m *Model) Cursor() int {
	return m.cursor
}

func (m *Model) SetCursor(index int) {
	if index >= 0 && index < len(m.rows) {
		m.cursor = index
		m.context.ContinueAction("@revisions.select")
	}
}

func (m *Model) Len() int {
	return len(m.rows)
}

var _ operations.SegmentRenderer = (*nullSegmentRenderer)(nil)

type nullSegmentRenderer struct{}

func (n nullSegmentRenderer) RenderSegment(currentStyle lipgloss.Style, segment *screen.Segment, row parser.Row) string {
	return currentStyle.Render(segment.Text)
}

func (m *Model) GetItemRenderer(index int) list.IItemRenderer {
	var (
		before, after, renderOverDescription, beforeCommitId, beforeChangeId string
	)
	row := m.rows[index]
	inLane := m.renderer.tracer.IsInSameLane(index)
	isHighlighted := index == m.cursor

	var op tea.Model
	if len(m.router.Views) > 0 {
		op = m.router.Views[m.router.Scope]
		if op, ok := op.(operations.Operation); ok {
			before = op.Render(row.Commit, operations.RenderPositionBefore)
			after = op.Render(row.Commit, operations.RenderPositionAfter)
			renderOverDescription = ""
			if isHighlighted {
				renderOverDescription = op.Render(row.Commit, operations.RenderOverDescription)
			}
			beforeCommitId = op.Render(row.Commit, operations.RenderBeforeCommitId)
			beforeChangeId = op.Render(row.Commit, operations.RenderBeforeChangeId)
		}
	}

	var segmentRenderer operations.SegmentRenderer
	if op != nil {
		if sr, ok := op.(operations.SegmentRenderer); ok {
			segmentRenderer = sr
		}
	}

	if segmentRenderer == nil {
		segmentRenderer = nullSegmentRenderer{}
	}

	return &itemRenderer{
		row:            row,
		before:         before,
		after:          after,
		description:    renderOverDescription,
		beforeChangeId: beforeChangeId,
		beforeCommitId: beforeCommitId,
		isHighlighted:  isHighlighted,
		SearchText:     m.quickSearch,
		textStyle:      m.textStyle,
		dimmedStyle:    m.dimmedStyle,
		selectedStyle:  m.selectedStyle,
		isChecked:      m.renderer.selections[row.Commit.GetChangeId()],
		isGutterInLane: func(lineIndex, segmentIndex int) bool {
			return m.renderer.tracer.IsGutterInLane(index, lineIndex, segmentIndex)
		},
		updateGutterText: func(lineIndex, segmentIndex int, text string) string {
			return m.renderer.tracer.UpdateGutterText(index, lineIndex, segmentIndex, text)
		},
		inLane:          inLane,
		segmentRenderer: segmentRenderer,
	}
}

type revisionsMsg struct {
	msg tea.Msg
}

// Allow a message to be targetted to this component.
func RevisionsCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return revisionsMsg{msg: msg}
	}
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

func (m *Model) ShortHelp() []key.Binding {
	if len(m.router.Views) > 0 {
		if status, ok := m.router.Views[m.router.Scope].(view.IStatus); ok {
			return status.ShortHelp()
		}
	}

	return []key.Binding{
		m.keymap.Up,
		m.keymap.Down,
		m.keymap.Quit,
		m.keymap.Help,
		m.keymap.Refresh,
		m.keymap.Preview.Mode,
		m.keymap.Revset,
		m.keymap.Details.Mode,
		m.keymap.Evolog.Mode,
		m.keymap.Rebase.Mode,
		m.keymap.Squash.Mode,
		m.keymap.Bookmark.Mode,
		m.keymap.Git.Mode,
		m.keymap.OpLog.Mode,
	}
}

func (m *Model) FullHelp() [][]key.Binding {
	if len(m.router.Views) > 0 {
		op := m.router.Views[m.router.Scope]
		if op, ok := op.(help.KeyMap); ok {
			return op.FullHelp()
		}
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
	if len(m.rows) == 0 || m.cursor == -1 {
		return jj.SelectedRevisions{}
	}
	var selected []*jj.Commit
	ids := make(map[string]bool)
	for commitId := range m.checkedRevisions {
		ids[commitId] = true
	}
	for _, row := range m.rows {
		if _, ok := ids[row.Commit.CommitId]; ok {
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

func (m *Model) getCurrentOp() tea.Model {
	if len(m.router.Views) > 0 {
		return m.router.Views[m.router.Scope]
	}
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(revisionsMsg); ok {
		msg = k.msg
	}

	var cmd tea.Cmd
	var nm *Model
	nm, cmd = m.internalUpdate(msg)

	if curSelected := m.SelectedRevision(); curSelected != nil {
		op := m.getCurrentOp()
		if op, ok := op.(operations.TracksSelectedRevision); ok {
			op.SetSelectedRevision(curSelected)
		}
	}

	return nm, cmd
}

func (m *Model) internalUpdate(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		switch msg.Action.Id {
		case "revisions.toggle_select":
			commit := m.rows[m.cursor].Commit
			changeId := commit.GetChangeId()
			if _, ok := m.checkedRevisions[changeId]; ok {
				delete(m.checkedRevisions, changeId)
			} else {
				m.checkedRevisions[changeId] = true
			}
			return m, nil
		case "open ace_jump":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(scopeAceJump, ace_jump.NewOperation(m, func(index int) parser.Row {
				return m.rows[index]
			}, m.renderer.FirstRowIndex, m.renderer.LastRowIndex))
			return m, cmd
		case "revisions.new":
			return m, m.context.RunCommand(jj.New(m.SelectedRevisions()), common.RefreshAndSelect("@"))
		case "revisions.commit":
			return m, m.context.RunInteractiveCommand(jj.CommitWorkingCopy(), common.Refresh)
		case "revisions.edit":
			ignoreImmutable := msg.Action.Get("ignore_immutable", false).(bool)
			return m, m.context.RunCommand(jj.Edit(m.SelectedRevision().GetChangeId(), ignoreImmutable), common.Refresh)
		case "revisions.diffedit":
			changeId := m.SelectedRevision().GetChangeId()
			return m, m.context.RunInteractiveCommand(jj.DiffEdit(changeId), common.Refresh)
		case "revisions.absorb":
			changeId := m.SelectedRevision().GetChangeId()
			return m, m.context.RunCommand(jj.Absorb(changeId), common.Refresh)
		case "open abandon":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(scopeAbandon, abandon.NewOperation(m.context, m.SelectedRevisions()))
			return m, cmd
		case "open set_bookmark":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(scopeSetBookmark, bookmark.NewSetBookmarkOperation(m.context, m.SelectedRevision().GetChangeId()))
			return m, cmd
		case "revisions.find":
			changeId := msg.Action.Get("change_id", "").(string)
			index := m.selectRevision(changeId)
			if index != -1 {
				m.SetCursor(index)
			}
			return m, nil
		case "revisions.jump_to_parent":
			m.jumpToParent(m.SelectedRevisions())
			return m, nil
		case "revisions.jump_to_children":
			immediate, _ := m.context.RunCommandImmediate(jj.GetFirstChild(m.SelectedRevision()))
			index := m.selectRevision(string(immediate))
			if index != -1 {
				m.SetCursor(index)
			}
			return m, nil
		case "refresh":
			return m, common.Refresh
		case "revisions.quick_search_cycle":
			m.SetCursor(m.search(m.cursor + 1))
			m.renderer.Reset()
			return m, nil
		case "revisions.diff":
			return m, tea.Sequence(actions.InvokeAction(actions.Action{Id: "open diff"}), func() tea.Msg {
				changeId := m.SelectedRevision().GetChangeId()
				output, _ := m.context.RunCommandImmediate(jj.Diff(changeId, ""))
				return common.ShowDiffMsg(output)
			})
		case "revisions.split":
			currentRevision := m.SelectedRevision().GetChangeId()
			return m, m.context.RunInteractiveCommand(jj.Split(currentRevision, []string{}), common.Refresh)
		case "revisions.describe":
			selections := m.SelectedRevisions()
			return m, m.context.RunInteractiveCommand(jj.Describe(selections), common.Refresh)
		case "revisions.revert":
			op := revert.NewOperation(m.context, m.SelectedRevisions(), revert.TargetDestination)
			return m, op.Init()
		case "open duplicate":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(scopeDuplicate, duplicate.NewOperation(m.context, m.SelectedRevisions(), duplicate.TargetDestination))
			return m, cmd
		case "open set_parents":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(scopeSetParents, set_parents.NewModel(m.context, m.SelectedRevision()))
			return m, cmd
		case "open evolog":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(scopeEvolog, evolog.NewOperation(m.context, m.SelectedRevision(), m.Width, m.Height))
			return m, cmd
		case "open inline_describe":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(scopeInlineDescribe, describe.NewOperation(m.context, m.SelectedRevision().GetChangeId(), m.Width))
			return m, cmd
		case "open revert":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(scopeRevert, revert.NewOperation(m.context, m.SelectedRevisions(), revert.TargetDestination))
			return m, cmd
		case "open squash":
			selectedRevisions := m.SelectedRevisions()
			parent, _ := m.context.RunCommandImmediate(jj.GetParent(selectedRevisions))
			parentIdx := m.selectRevision(string(parent))
			if parentIdx != -1 {
				m.SetCursor(parentIdx)
			} else if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
			files := msg.Action.Get("files", []string{}).([]string)
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(scopeSquash, squash.NewOperation(m.context, selectedRevisions, squash.WithFiles(files)))
			return m, cmd
		case "open details":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(scopeDetails, details.NewOperation(m.context, m.SelectedRevision(), m.Height))
			return m, cmd
		case "open rebase":
			var cmd tea.Cmd
			m.router, cmd = m.router.Open(scopeRebase, rebase.NewOperation(m.context, m.SelectedRevisions(), rebase.SourceRevision, rebase.TargetDestination))
			return m, cmd
		case "revisions.up":
			if m.cursor >= 1 {
				m.SetCursor(m.cursor - 1)
				log.Println("action: revisions.up handled ", m.cursor)
				return m, nil
			}
			return m, nil
		case "revisions.down":
			if m.cursor+1 < len(m.rows) {
				m.SetCursor(m.cursor + 1)
				return m, nil
			} else if m.hasMore {
				return m, m.requestMoreRows(m.tag.Load())
			}
			return m, nil
		}
	case common.QuickSearchMsg:
		m.quickSearch = string(msg)
		m.SetCursor(m.search(0))
		m.renderer.Reset()
		return m, nil
	case common.CommandCompletedMsg:
		m.output = msg.Output
		m.err = msg.Err
		return m, nil
	case common.RefreshMsg:
		if !msg.KeepSelections {
			m.checkedRevisions = make(map[string]bool)
		}
		m.isLoading = true
		var cmd tea.Cmd
		m.router, cmd = m.router.Update(msg)
		if config.Current.Revisions.LogBatching {
			currentTag := m.tag.Add(1)
			return m, tea.Batch(m.loadStreaming(m.context.CurrentRevset, msg.SelectedRevision, currentTag), cmd)
		} else {
			return m, tea.Batch(m.load(m.context.CurrentRevset, msg.SelectedRevision), cmd)
		}
	case updateRevisionsMsg:
		m.isLoading = false
		m.updateGraphRows(msg.rows, msg.selectedRevision)
		return m, tea.Batch(m.highlightChanges, func() tea.Msg {
			return common.UpdateRevisionsSuccessMsg{}
		})
	case startRowsStreamingMsg:
		m.offScreenRows = nil
		m.revisionToSelect = msg.selectedRevision

		// If the revision to select is not set, use the currently selected item
		if m.revisionToSelect == "" {
			if current := m.SelectedRevision(); current != nil {
				m.revisionToSelect = current.GetChangeId()
			}
		}
		log.Println("Starting streaming revisions message received with tag:", msg.tag, "revision to select:", msg.selectedRevision)
		return m, m.requestMoreRows(msg.tag)
	case appendRowsBatchMsg:
		if msg.tag != m.tag.Load() {
			return m, nil
		}
		m.offScreenRows = append(m.offScreenRows, msg.rows...)
		m.hasMore = msg.hasMore
		m.isLoading = m.hasMore && len(m.offScreenRows) > 0

		if m.hasMore {
			// keep requesting rows until we reach the initial load count or the current cursor position
			if len(m.offScreenRows) < m.cursor+1 || len(m.offScreenRows) < m.renderer.ViewRange.LastRowIndex+1 {
				return m, m.requestMoreRows(msg.tag)
			}
		} else if m.streamer != nil {
			m.streamer.Close()
		}

		currentSelectedRevision := m.SelectedRevision()
		m.rows = m.offScreenRows
		if m.revisionToSelect != "" {
			m.SetCursor(m.selectRevision(m.revisionToSelect))
			m.revisionToSelect = ""
		}

		if m.cursor == -1 && currentSelectedRevision != nil {
			m.SetCursor(m.selectRevision(currentSelectedRevision.GetChangeId()))
		}

		if (m.cursor < 0 || m.cursor >= len(m.rows)) && len(m.rows) > 0 {
			m.SetCursor(0)
		}

		m.context.ContinueAction("@refresh")
		cmds := []tea.Cmd{m.highlightChanges}
		if !m.hasMore {
			cmds = append(cmds, func() tea.Msg {
				return common.UpdateRevisionsSuccessMsg{}
			})
		}
		return m, tea.Batch(cmds...)
	}

	if len(m.rows) == 0 {
		return m, nil
	}

	var cmd tea.Cmd
	m.router, cmd = m.router.Update(msg)
	return m, cmd
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
			m.SetCursor(m.selectRevision("@"))
		}
		if m.cursor == -1 {
			m.SetCursor(0)
		}
	} else {
		m.SetCursor(0)
	}
}

func (m *Model) View() string {
	if len(m.rows) == 0 {
		if m.isLoading {
			return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, "loading")
		}
		return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, "(no matching revisions)")
	}

	if config.Current.UI.Tracer.Enabled {
		start, end := m.renderer.FirstRowIndex, m.renderer.LastRowIndex+1 // +1 because the last row is inclusive in the view range
		//log.Println("Visible row range:", start, end, "Cursor:", m.cursor, "Total rows:", len(m.rows))
		m.renderer.tracer = parser.NewTracer(m.rows, m.cursor, start, end)
	} else {
		m.renderer.tracer = parser.NewNoopTracer()
	}

	m.renderer.selections = m.checkedRevisions

	output := m.renderer.Render(m.cursor)
	output = m.textStyle.MaxWidth(m.Width).Render(output)
	return lipgloss.Place(m.Width, m.Height, 0, 0, output)
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
	if m.tag.Load() != tag {
		return nil
	}

	if m.streamer != nil {
		m.streamer.Close()
		m.streamer = nil
	}

	m.hasMore = false

	var notifyErrorCmd tea.Cmd
	streamer, err := graph.NewGraphStreamer(m.context, revset)
	if err != nil {
		notifyErrorCmd = func() tea.Msg {
			return common.UpdateRevisionsFailedMsg{
				Err:    err,
				Output: fmt.Sprintf("%v", err),
			}
		}
	}
	m.streamer = streamer
	m.hasMore = true
	m.offScreenRows = nil
	log.Println("Starting streaming revisions with tag:", tag)
	startStreamingCmd := func() tea.Msg {
		return startRowsStreamingMsg{selectedRevision, tag}
	}

	return tea.Batch(startStreamingCmd, notifyErrorCmd)
}

func (m *Model) requestMoreRows(tag uint64) tea.Cmd {
	return func() tea.Msg {
		if m.streamer == nil || !m.hasMore {
			return nil
		}
		if tag == m.tag.Load() {
			batch := m.streamer.RequestMore()
			return appendRowsBatchMsg{batch.Rows, batch.HasMore, tag}
		}
		return nil
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

func (m *Model) GetCommitIds() []string {
	var commitIds []string
	for _, row := range m.rows {
		commitIds = append(commitIds, row.Commit.CommitId)
	}
	return commitIds
}

func New(c *appContext.MainContext) *Model {
	keymap := config.Current.GetKeyMap()
	router := view.NewRouter(c, "")
	m := Model{
		Sizeable:         &common.Sizeable{Width: 0, Height: 0},
		context:          c,
		keymap:           keymap,
		rows:             nil,
		offScreenRows:    nil,
		cursor:           0,
		textStyle:        common.DefaultPalette.Get("revisions text"),
		dimmedStyle:      common.DefaultPalette.Get("revisions dimmed"),
		selectedStyle:    common.DefaultPalette.Get("revisions selected"),
		checkedRevisions: make(map[string]bool),
		router:           router,
	}
	m.renderer = newRevisionListRenderer(&m, m.Sizeable)
	return &m
}

func (m *Model) jumpToParent(revisions jj.SelectedRevisions) {
	immediate, _ := m.context.RunCommandImmediate(jj.GetParent(revisions))
	parentIndex := m.selectRevision(string(immediate))
	if parentIndex != -1 {
		m.SetCursor(parentIndex)
	}
}
