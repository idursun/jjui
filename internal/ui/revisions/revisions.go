package revisions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations/ace_jump"
	"github.com/idursun/jjui/internal/ui/operations/duplicate"
	"github.com/idursun/jjui/internal/ui/operations/revert"
	"github.com/idursun/jjui/internal/ui/operations/set_parents"
	"github.com/idursun/jjui/internal/ui/render"

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

var (
	_ common.Focusable      = (*Model)(nil)
	_ common.Editable       = (*Model)(nil)
	_ common.ImmediateModel = (*Model)(nil)
)

type Model struct {
	rows                   []parser.Row
	tag                    atomic.Uint64
	revisionToSelect       string
	offScreenRows          []parser.Row
	streamer               *graph.GraphStreamer
	hasMore                bool
	op                     common.ImmediateModel
	cursor                 int
	context                *appContext.MainContext
	keymap                 config.KeyMappings[key.Binding]
	output                 string
	err                    error
	quickSearch            string
	previousOpLogId        string
	isLoading              bool
	displayContextRenderer *DisplayContextRenderer
	textStyle              lipgloss.Style
	dimmedStyle            lipgloss.Style
	selectedStyle          lipgloss.Style
	matchedStyle           lipgloss.Style
	ensureCursorView       bool
	requestInFlight        bool
}

type revisionsMsg struct {
	msg tea.Msg
}

func RevisionsCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return revisionsMsg{msg: msg}
	}
}

type ItemClickedMsg struct {
	Index int
}

type ViewportScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (v ViewportScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	v.Delta = delta
	v.Horizontal = horizontal
	return v
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

func (m *Model) Cursor() int {
	return m.cursor
}

func (m *Model) SetCursor(index int) {
	if index >= 0 && index < len(m.rows) {
		m.cursor = index
		m.ensureCursorView = true
	}
}

func (m *Model) HasMore() bool {
	return m.hasMore
}

func (m *Model) Scroll(delta int) tea.Cmd {
	m.ensureCursorView = false
	currentStart := m.displayContextRenderer.GetScrollOffset()
	desiredStart := currentStart + delta
	m.displayContextRenderer.SetScrollOffset(desiredStart)

	// Request more rows if scrolling down and near the end
	if m.hasMore && delta > 0 {
		lastRowIndex := m.displayContextRenderer.GetLastRowIndex()
		if lastRowIndex >= len(m.rows)-1 {
			return m.requestMoreRows(m.tag.Load())
		}
	}
	return nil
}

func (m *Model) Len() int {
	return len(m.rows)
}

func (m *Model) IsEditing() bool {
	if f, ok := m.op.(common.Editable); ok {
		return f.IsEditing()
	}
	return false
}

func (m *Model) IsFocused() bool {
	if f, ok := m.op.(common.Focusable); ok {
		return f.IsFocused()
	}
	return false
}

func (m *Model) InNormalMode() bool {
	if _, ok := m.op.(*operations.Default); ok {
		return true
	}
	return false
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
	ids := make(map[string]bool)
	for _, ci := range m.context.CheckedItems {
		if rev, ok := ci.(appContext.SelectedRevision); ok {
			ids[rev.CommitId] = true
		}
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

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(revisionsMsg); ok {
		msg = k.msg
	}
	cmd := m.internalUpdate(msg)

	if curSelected := m.SelectedRevision(); curSelected != nil {
		if op, ok := m.op.(operations.TracksSelectedRevision); ok {
			cmd = tea.Batch(cmd, op.SetSelectedRevision(curSelected))
		}
	}

	return cmd
}

func (m *Model) internalUpdate(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		return m.handleIntent(msg)
	case ItemClickedMsg:
		if !m.InNormalMode() {
			return nil
		}
		m.SetCursor(msg.Index)
		return m.updateSelection()
	case ViewportScrollMsg:
		if msg.Horizontal {
			return nil
		}
		return m.Scroll(msg.Delta)

	case common.CloseViewMsg:
		m.op = operations.NewDefault()
		return m.updateSelection()
	case common.RestoreOperationMsg:
		if op, ok := msg.Operation.(operations.Operation); ok {
			m.op = op
			return m.updateSelection()
		}
		m.op = operations.NewDefault()
		return m.updateSelection()
	case common.QuickSearchMsg:
		m.quickSearch = strings.ToLower(string(msg))
		m.SetCursor(m.search(0))
		m.op = operations.NewDefault()
		return nil
	case common.CommandCompletedMsg:
		m.output = msg.Output
		m.err = msg.Err
		return nil
	case common.AutoRefreshMsg:
		id, _ := m.context.RunCommandImmediate(jj.OpLogId(true))
		currentOperationId := string(id)
		log.Println("Previous operation ID:", m.previousOpLogId, "Current operation ID:", currentOperationId)
		if currentOperationId != m.previousOpLogId {
			m.previousOpLogId = currentOperationId
			return common.RefreshAndKeepSelections
		}
	case common.UpdateRevisionsFailedMsg:
		m.isLoading = false
		return nil
	case common.RefreshMsg:
		return tea.Batch(m.refresh(intents.Refresh{
			KeepSelections:   msg.KeepSelections,
			SelectedRevision: msg.SelectedRevision,
		}), m.op.Update(msg))
	case updateRevisionsMsg:
		m.isLoading = false
		m.updateGraphRows(msg.rows, msg.selectedRevision)
		return tea.Batch(m.highlightChanges, m.updateSelection(), func() tea.Msg {
			return common.UpdateRevisionsSuccessMsg{}
		})
	case startRowsStreamingMsg:
		m.offScreenRows = nil
		m.revisionToSelect = msg.selectedRevision

		// If the revision to select is not set, use the currently selected item
		if m.revisionToSelect == "" {
			switch selected := m.context.SelectedItem.(type) {
			case appContext.SelectedRevision:
				m.revisionToSelect = selected.CommitId
			case appContext.SelectedFile:
				m.revisionToSelect = selected.CommitId
			}
		}
		log.Println("Starting streaming revisions with tag:", msg.tag)
		return m.requestMoreRows(msg.tag)
	case appendRowsBatchMsg:
		m.requestInFlight = false

		if msg.tag != m.tag.Load() {
			return nil
		}
		m.offScreenRows = append(m.offScreenRows, msg.rows...)
		m.hasMore = msg.hasMore
		m.isLoading = m.hasMore && len(m.offScreenRows) > 0

		if m.hasMore {
			// keep requesting rows until we reach the initial load count or the current cursor position
			lastRowIndex := m.displayContextRenderer.GetLastRowIndex()
			if len(m.offScreenRows) < m.cursor+1 || len(m.offScreenRows) < lastRowIndex+1 {
				return m.requestMoreRows(msg.tag)
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

		cmds := []tea.Cmd{m.highlightChanges, m.updateSelection()}
		if len(m.offScreenRows) > 0 {
			cmds = append(cmds, func() tea.Msg {
				return common.UpdateRevisionsSuccessMsg{}
			})
		}
		return tea.Batch(cmds...)
	}

	// Non-input messages are broadcast to the current operation
	if !common.IsInputMessage(msg) {
		return m.op.Update(msg)
	}

	if len(m.rows) == 0 {
		return nil
	}

	if op, ok := m.op.(common.Editable); ok && op.IsEditing() {
		return m.op.Update(msg)
	}

	if op, ok := m.op.(common.Overlay); ok && op.IsOverlay() {
		return m.op.Update(msg)
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Up, m.keymap.ScrollUp):
			return m.handleIntent(intents.Navigate{Delta: -1, IsPage: key.Matches(msg, m.keymap.ScrollUp)})
		case key.Matches(msg, m.keymap.Down, m.keymap.ScrollDown):
			return m.handleIntent(intents.Navigate{Delta: 1, IsPage: key.Matches(msg, m.keymap.ScrollDown)})
		case key.Matches(msg, m.keymap.JumpToParent):
			return m.handleIntent(intents.Navigate{Target: intents.TargetParent})
		case key.Matches(msg, m.keymap.JumpToChildren):
			return m.handleIntent(intents.Navigate{Target: intents.TargetChild})
		case key.Matches(msg, m.keymap.JumpToWorkingCopy):
			return m.handleIntent(intents.Navigate{Target: intents.TargetWorkingCopy})
		case key.Matches(msg, m.keymap.AceJump):
			parentOp := m.op
			// Create ace jump with parent operation
			op := ace_jump.NewOperation(m.SetCursor, func(index int) parser.Row {
				return m.rows[index]
			}, m.displayContextRenderer.GetFirstRowIndex(), m.displayContextRenderer.GetLastRowIndex(), parentOp)
			m.op = op
			return op.Init()
		default:
			if op, ok := m.op.(common.Focusable); ok && op.IsFocused() {
				return m.op.Update(msg)
			}

			switch {
			case m.quickSearch != "" && (msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter):
				m.quickSearch = ""
				return nil
			case key.Matches(msg, m.keymap.ToggleSelect):
				commit := m.rows[m.cursor].Commit
				changeId := commit.GetChangeId()
				item := appContext.SelectedRevision{ChangeId: changeId, CommitId: commit.CommitId}
				m.context.ToggleCheckedItem(item)
				m.jumpToParent(jj.NewSelectedRevisions(commit))
			case key.Matches(msg, m.keymap.Cancel):
				m.context.ClearCheckedItems(reflect.TypeFor[appContext.SelectedRevision]())
				m.op = operations.NewDefault()
				return nil
			case key.Matches(msg, m.keymap.QuickSearchCycle):
				m.SetCursor(m.search(m.cursor + 1))
				return nil
			case key.Matches(msg, m.keymap.Details.Mode):
				return m.handleIntent(intents.OpenDetails{})
			case key.Matches(msg, m.keymap.InlineDescribe.Mode):
				return m.handleIntent(intents.StartInlineDescribe{})
			case key.Matches(msg, m.keymap.New):
				return m.handleIntent(intents.StartNew{})
			case key.Matches(msg, m.keymap.Commit):
				return m.handleIntent(intents.CommitWorkingCopy{})
			case key.Matches(msg, m.keymap.Edit, m.keymap.ForceEdit):
				ignoreImmutable := key.Matches(msg, m.keymap.ForceEdit)
				return m.handleIntent(intents.StartEdit{IgnoreImmutable: ignoreImmutable})
			case key.Matches(msg, m.keymap.Diffedit):
				return m.handleIntent(intents.StartDiffEdit{})
			case key.Matches(msg, m.keymap.Absorb):
				return m.handleIntent(intents.StartAbsorb{})
			case key.Matches(msg, m.keymap.Abandon):
				return m.handleIntent(intents.StartAbandon{})
			case key.Matches(msg, m.keymap.Bookmark.Set):
				m.op = bookmark.NewSetBookmarkOperation(m.context, m.SelectedRevision().GetChangeId())
				return m.op.Init()
			case key.Matches(msg, m.keymap.Split, m.keymap.SplitParallel):
				return m.handleIntent(intents.StartSplit{
					IsParallel: key.Matches(msg, m.keymap.SplitParallel),
				})
			case key.Matches(msg, m.keymap.Describe):
				return m.handleIntent(intents.StartDescribe{})
			case key.Matches(msg, m.keymap.Evolog.Mode):
				return m.handleIntent(intents.StartEvolog{})
			case key.Matches(msg, m.keymap.Diff):
				return m.handleIntent(intents.ShowDiff{})
			case key.Matches(msg, m.keymap.Refresh):
				return m.handleIntent(intents.Refresh{})
			case key.Matches(msg, m.keymap.Squash.Mode):
				return m.handleIntent(intents.StartSquash{})
			case key.Matches(msg, m.keymap.Revert.Mode):
				return m.handleIntent(intents.StartRevert{})
			case key.Matches(msg, m.keymap.Rebase.Mode):
				return m.handleIntent(intents.StartRebase{})
			case key.Matches(msg, m.keymap.Duplicate.Mode):
				return m.handleIntent(intents.StartDuplicate{})
			case key.Matches(msg, m.keymap.SetParents):
				return m.handleIntent(intents.SetParents{})
			}
		}
	}

	return cmd
}

func (m *Model) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent := intent.(type) {
	case intents.OpenDetails:
		return m.openDetails(intent)
	case intents.StartSquash:
		return m.startSquash(intent)
	case intents.StartInlineDescribe:
		return m.startInlineDescribe(intent)
	case intents.StartAbsorb:
		return m.startAbsorb(intent)
	case intents.StartAbandon:
		return m.startAbandon(intent)
	case intents.StartNew:
		return m.startNew(intent)
	case intents.CommitWorkingCopy:
		return m.commitWorkingCopy()
	case intents.StartEdit:
		return m.startEdit(intent)
	case intents.StartDiffEdit:
		return m.startDiffEdit(intent)
	case intents.StartRevert:
		return m.startRevert(intent)
	case intents.StartDuplicate:
		return m.startDuplicate(intent)
	case intents.SetParents:
		return m.startSetParents(intent)
	case intents.Navigate:
		return m.navigate(intent)
	case intents.StartDescribe:
		return m.startDescribe(intent)
	case intents.StartEvolog:
		return m.startEvolog(intent)
	case intents.ShowDiff:
		return m.showDiff(intent)
	case intents.StartSplit:
		return m.startSplit(intent)
	case intents.StartRebase:
		return m.startRebase(intent)
	case intents.Refresh:
		return m.refresh(intent)
	}
	return nil
}

func (m *Model) refresh(intent intents.Refresh) tea.Cmd {
	if !intent.KeepSelections {
		m.context.ClearCheckedItems(reflect.TypeFor[appContext.SelectedRevision]())
	}
	m.isLoading = true
	if config.Current.Revisions.LogBatching {
		currentTag := m.tag.Add(1)
		return m.loadStreaming(m.context.CurrentRevset, intent.SelectedRevision, currentTag)
	}
	return m.load(m.context.CurrentRevset, intent.SelectedRevision)
}

func (m *Model) openDetails(_ intents.OpenDetails) tea.Cmd {
	if m.SelectedRevision() == nil {
		return nil
	}
	model := details.NewOperation(m.context, m.SelectedRevision())
	m.op = model
	return m.op.Init()
}

func (m *Model) startSquash(intent intents.StartSquash) tea.Cmd {
	selected := intent.Selected
	if len(selected.Revisions) == 0 {
		selected = m.SelectedRevisions()
	}
	if len(selected.Revisions) == 0 {
		return nil
	}

	parent, _ := m.context.RunCommandImmediate(jj.GetParent(selected))
	parentIdx := m.selectRevision(string(parent))
	if parentIdx != -1 {
		m.SetCursor(parentIdx)
	} else if m.cursor < len(m.rows)-1 {
		m.SetCursor(m.cursor + 1)
	}
	m.op = squash.NewOperation(m.context, selected, squash.WithFiles(intent.Files))
	return m.op.Init()
}

func (m *Model) startRebase(intent intents.StartRebase) tea.Cmd {
	selected := intent.Selected
	if len(selected.Revisions) == 0 {
		selected = m.SelectedRevisions()
	}
	if len(selected.Revisions) == 0 {
		return nil
	}

	source := intent.Source
	if source == 0 {
		source = rebase.SourceRevision
	}
	target := intent.Target
	if target == 0 {
		target = rebase.TargetDestination
	}
	m.op = rebase.NewOperation(m.context, selected, source, target)
	return m.op.Init()
}

func (m *Model) startRevert(intent intents.StartRevert) tea.Cmd {
	selected := intent.Selected
	if len(selected.Revisions) == 0 {
		selected = m.SelectedRevisions()
	}
	if len(selected.Revisions) == 0 {
		return nil
	}

	target := intent.Target
	if target == 0 {
		target = revert.TargetDestination
	}
	m.op = revert.NewOperation(m.context, selected, target)
	return m.op.Init()
}

func (m *Model) startDuplicate(intent intents.StartDuplicate) tea.Cmd {
	selected := intent.Selected
	if len(selected.Revisions) == 0 {
		selected = m.SelectedRevisions()
	}
	if len(selected.Revisions) == 0 {
		return nil
	}

	m.op = duplicate.NewOperation(m.context, selected, duplicate.TargetDestination)
	return m.op.Init()
}

func (m *Model) startSetParents(intent intents.SetParents) tea.Cmd {
	commit := intent.Selected
	if commit == nil {
		commit = m.SelectedRevision()
	}
	if commit == nil {
		return nil
	}

	m.op = set_parents.NewModel(m.context, commit)
	return m.op.Init()
}

func (m *Model) startNew(intent intents.StartNew) tea.Cmd {
	selected := intent.Selected
	if len(selected.Revisions) == 0 {
		selected = m.SelectedRevisions()
	}
	return m.context.RunCommand(jj.New(selected), common.RefreshAndSelect("@"))
}

func (m *Model) commitWorkingCopy() tea.Cmd {
	return m.context.RunInteractiveCommand(jj.CommitWorkingCopy(), common.Refresh)
}

func (m *Model) startEdit(intent intents.StartEdit) tea.Cmd {
	commit := intent.Selected
	if commit == nil {
		commit = m.SelectedRevision()
	}
	if commit == nil {
		return nil
	}
	return m.context.RunCommand(jj.Edit(commit.GetChangeId(), intent.IgnoreImmutable), common.Refresh)
}

func (m *Model) startDiffEdit(intent intents.StartDiffEdit) tea.Cmd {
	commit := intent.Selected
	if commit == nil {
		commit = m.SelectedRevision()
	}
	if commit == nil {
		return nil
	}
	return m.context.RunInteractiveCommand(jj.DiffEdit(commit.GetChangeId()), common.Refresh)
}

func (m *Model) startAbsorb(intent intents.StartAbsorb) tea.Cmd {
	commit := intent.Selected
	if commit == nil {
		commit = m.SelectedRevision()
	}
	if commit == nil {
		return nil
	}
	return m.context.RunCommand(jj.Absorb(commit.GetChangeId()), common.Refresh)
}

func (m *Model) startAbandon(intent intents.StartAbandon) tea.Cmd {
	selected := intent.Selected
	if len(selected.Revisions) == 0 {
		selected = m.SelectedRevisions()
	}
	if len(selected.Revisions) == 0 {
		return nil
	}
	m.op = abandon.NewOperation(m.context, selected)
	return m.op.Init()
}

func (m *Model) navigate(intent intents.Navigate) tea.Cmd {
	if len(m.rows) == 0 {
		return nil
	}

	ensureView := true
	if intent.EnsureView != nil {
		ensureView = *intent.EnsureView
	}
	allowStream := true
	if intent.AllowStream != nil {
		allowStream = *intent.AllowStream
	}

	if intent.ChangeID != "" || intent.FallbackID != "" {
		idx := m.selectRevision(intent.ChangeID)
		if idx == -1 && intent.FallbackID != "" {
			idx = m.selectRevision(intent.FallbackID)
		}
		if idx == -1 {
			return nil
		}
		m.ensureCursorView = ensureView
		m.SetCursor(idx)
		return m.updateSelection()
	}

	switch intent.Target {
	case intents.TargetParent:
		m.jumpToParent(m.SelectedRevisions())
		m.ensureCursorView = ensureView
		return m.updateSelection()
	case intents.TargetWorkingCopy:
		if idx := m.selectRevision("@"); idx != -1 {
			m.SetCursor(idx)
		}
		m.ensureCursorView = ensureView
		return m.updateSelection()
	case intents.TargetChild:
		immediate, _ := m.context.RunCommandImmediate(jj.GetFirstChild(m.SelectedRevision()))
		if idx := m.selectRevision(string(immediate)); idx != -1 {
			m.SetCursor(idx)
		}
		m.ensureCursorView = ensureView
		return m.updateSelection()
	}

	delta := intent.Delta
	if delta == 0 {
		delta = 1
	}

	// Calculate step (convert page scroll to item count)
	step := delta
	if intent.IsPage {
		firstRowIndex := m.displayContextRenderer.GetFirstRowIndex()
		lastRowIndex := m.displayContextRenderer.GetLastRowIndex()
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

	if step > 0 {
		// Moving down
		if newCursor >= totalItems {
			if allowStream && m.hasMore {
				return m.requestMoreRows(m.tag.Load())
			}
			newCursor = totalItems - 1
		}
	} else {
		// Moving up
		if newCursor < 0 {
			newCursor = 0
		}
	}

	m.SetCursor(newCursor)
	m.ensureCursorView = ensureView
	return m.updateSelection()
}

func (m *Model) startDescribe(intent intents.StartDescribe) tea.Cmd {
	selected := intent.Selected
	if len(selected.Revisions) == 0 {
		selected = m.SelectedRevisions()
	}
	if len(selected.Revisions) == 0 {
		return nil
	}
	return m.context.RunInteractiveCommand(jj.Describe(selected), common.Refresh)
}

func (m *Model) startEvolog(intent intents.StartEvolog) tea.Cmd {
	commit := intent.Selected
	if commit == nil {
		commit = m.SelectedRevision()
	}
	if commit == nil {
		return nil
	}
	model := evolog.NewOperation(m.context, commit)
	m.op = model
	return m.op.Init()
}

func (m *Model) startInlineDescribe(intent intents.StartInlineDescribe) tea.Cmd {
	commit := intent.Selected
	if commit == nil {
		commit = m.SelectedRevision()
	}
	if commit == nil {
		return nil
	}
	model := describe.NewOperation(m.context, commit)
	m.op = model
	return m.op.Init()
}

func (m *Model) showDiff(intent intents.ShowDiff) tea.Cmd {
	commit := intent.Selected
	if commit == nil {
		commit = m.SelectedRevision()
	}
	if commit == nil {
		return nil
	}
	changeId := commit.GetChangeId()
	return func() tea.Msg {
		output, _ := m.context.RunCommandImmediate(jj.Diff(changeId, ""))
		return common.ShowDiffMsg(output)
	}
}

func (m *Model) startSplit(intent intents.StartSplit) tea.Cmd {
	commit := intent.Selected
	if commit == nil {
		commit = m.SelectedRevision()
	}
	if commit == nil {
		return nil
	}
	return m.context.RunInteractiveCommand(jj.Split(commit.GetChangeId(), intent.Files, intent.IsParallel), common.Refresh)
}

func (m *Model) updateSelection() tea.Cmd {
	// Don't override file-level selections (from Details panel)
	if _, isFile := m.context.SelectedItem.(appContext.SelectedFile); isFile && !m.InNormalMode() {
		return nil
	}
	if selectedRevision := m.SelectedRevision(); selectedRevision != nil {
		return m.context.SetSelectedItem(appContext.SelectedRevision{
			ChangeId: selectedRevision.GetChangeId(),
			CommitId: selectedRevision.CommitId,
		})
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
				if strings.HasPrefix(parts[0], row.Commit.GetChangeId()) {
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
		m.SetCursor(m.selectRevision(currentSelectedRevision))
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

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	area := box.R

	if len(m.rows) == 0 {
		content := ""
		if m.isLoading {
			content = lipgloss.Place(area.Dx(), area.Dy(), lipgloss.Center, lipgloss.Center, "loading")
		} else {
			content = lipgloss.Place(area.Dx(), area.Dy(), lipgloss.Center, lipgloss.Center, "(no matching revisions)")
		}
		dl.AddDraw(area, content, 0)
		return
	}

	// Set selections
	m.displayContextRenderer.SetSelections(m.context.GetSelectedRevisions())

	// Get operation if any
	var op operations.Operation
	if opModel, ok := m.op.(operations.Operation); ok {
		op = opModel
	}

	// Render to DisplayContext
	m.displayContextRenderer.Render(
		dl,
		m.rows,
		m.cursor,
		layout.Box{R: area},
		op,
		m.ensureCursorView,
	)

	// Reset the flag after ensuring cursor is visible
	m.ensureCursorView = false
}

func (m *Model) load(revset string, selectedRevision string) tea.Cmd {
	return func() tea.Msg {
		output, err := m.context.RunCommandImmediate(jj.Log(revset, config.Current.Limit, m.context.JJConfig.Templates.Log))
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

	var cmds []tea.Cmd
	streamer, err := graph.NewGraphStreamer(context.Background(), m.context, revset, m.context.JJConfig.Templates.Log)
	if err != nil {
		var errMsg string
		if err == io.EOF {
			errMsg = fmt.Sprintf("No revisions found for revset `%s`", revset)
			err = errors.New(errMsg)
		} else {
			errMsg = fmt.Sprintf("%v", err)
		}

		cmds = append(cmds, func() tea.Msg {
			return common.UpdateRevisionsFailedMsg{
				Err:    err,
				Output: errMsg,
			}
		})
	}

	if m.streamer != nil {
		m.streamer.Close()
		m.streamer = nil
	}
	m.streamer = streamer
	m.hasMore = true
	cmds = append(cmds, m.Update(startRowsStreamingMsg{selectedRevision: selectedRevision, tag: tag}))
	return tea.Batch(cmds...)
}

func (m *Model) requestMoreRows(tag uint64) tea.Cmd {
	if m.requestInFlight || m.streamer == nil || !m.hasMore || tag != m.tag.Load() {
		return nil
	}

	m.requestInFlight = true
	batch := m.streamer.RequestMore()
	return m.Update(appendRowsBatchMsg{batch.Rows, batch.HasMore, tag})
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
				if segment.Text != "" && strings.Contains(strings.ToLower(segment.Text), m.quickSearch) {
					return c
				}
			}
		}
	}
	return m.cursor
}

func (m *Model) CurrentOperation() operations.Operation {
	return m.op.(operations.Operation)
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
	m := Model{
		context:       c,
		keymap:        keymap,
		rows:          nil,
		offScreenRows: nil,
		op:            operations.NewDefault(),
		cursor:        0,
		textStyle:     common.DefaultPalette.Get("revisions text"),
		dimmedStyle:   common.DefaultPalette.Get("revisions dimmed"),
		selectedStyle: common.DefaultPalette.Get("revisions selected"),
		matchedStyle:  common.DefaultPalette.Get("revisions matched"),
	}
	m.displayContextRenderer = NewDisplayContextRenderer(m.textStyle, m.dimmedStyle, m.selectedStyle, m.matchedStyle)
	return &m
}

func (m *Model) jumpToParent(revisions jj.SelectedRevisions) {
	immediate, _ := m.context.RunCommandImmediate(jj.GetParent(revisions))
	parentIndex := m.selectRevision(string(immediate))
	if parentIndex != -1 {
		m.SetCursor(parentIndex)
	}
}
