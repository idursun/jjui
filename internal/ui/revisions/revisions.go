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

	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations/ace_jump"
	"github.com/idursun/jjui/internal/ui/operations/duplicate"
	"github.com/idursun/jjui/internal/ui/operations/revert"
	"github.com/idursun/jjui/internal/ui/operations/set_parents"
	"github.com/idursun/jjui/internal/ui/render"

	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/operations/describe"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	"github.com/idursun/jjui/internal/ui/routing"
)

var (
	_ common.Focusable      = (*Model)(nil)
	_ common.Editable       = (*Model)(nil)
	_ common.ImmediateModel = (*Model)(nil)
	_ routing.LayerProvider = (*Model)(nil)
)

const scopeQuickSearch bindings.Scope = "revisions.quick_search"

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
	Ctrl  bool
	Alt   bool
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

type streamingReadyMsg struct {
	streamer         *graph.GraphStreamer
	selectedRevision string
	tag              uint64
	err              error
	output           string
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

func (m *Model) IsOverlay() bool {
	if o, ok := m.op.(common.Overlay); ok {
		return o.IsOverlay()
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

func (m *Model) HasQuickSearch() bool {
	return m.quickSearch != ""
}

func (m *Model) Layers() []routing.Layer {
	var ret []routing.Layer
	if lp, ok := m.op.(routing.LayerProvider); ok {
		ret = append(ret, lp.Layers()...)
	}

	if m.quickSearch != "" {
		ret = append(ret, routing.Layer{
			Scope:     scopeQuickSearch,
			AllowLeak: !m.IsEditing() && !m.IsFocused(),
			Handler:   m,
		})
	}

	var scope bindings.Scope = actions.ScopeRevisions
	if !m.InNormalMode() {
		scope = ""
	}

	ret = append(ret, routing.Layer{
		Scope:     scope,
		AllowLeak: !m.IsEditing() && !m.IsFocused(),
		Handler:   m,
	})
	return ret
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
		if cmd, handled := m.HandleIntent(msg); handled {
			return cmd
		}
		return m.op.Update(msg)
	case ItemClickedMsg:
		// Don't allow changing selection if the operation is editing (e.g. describe)
		if editable, ok := m.op.(common.Editable); ok && editable.IsEditing() {
			return nil
		}
		// Don't allow changing selection if the operation is an overlay (e.g. details)
		if overlay, ok := m.op.(common.Overlay); ok && overlay.IsOverlay() {
			return nil
		}
		switch {
		case msg.Alt:
			m.rangeSelect(msg.Index)
		case msg.Ctrl:
			m.SetCursor(msg.Index)
			if commit := m.rows[msg.Index].Commit; commit != nil {
				item := appContext.SelectedRevision{ChangeId: commit.GetChangeId(), CommitId: commit.CommitId}
				m.context.ToggleCheckedItem(item)
			}
		default:
			m.SetCursor(msg.Index)
		}
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
	case common.StartAceJumpMsg:
		cmd, _ := m.HandleIntent(intents.StartAceJump{})
		return cmd
	case common.QuickSearchMsg:
		m.quickSearch = strings.ToLower(string(msg))
		m.SetCursor(m.search(0, false))
		m.op = operations.NewDefault()
		return m.updateSelection()
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
	case streamingReadyMsg:
		if msg.tag != m.tag.Load() {
			if msg.streamer != nil {
				msg.streamer.Close()
			}
			return nil
		}

		if m.streamer != nil {
			m.streamer.Close()
			m.streamer = nil
		}

		if msg.err != nil && msg.streamer == nil {
			m.hasMore = false
			m.isLoading = false
			m.requestInFlight = false
			return func() tea.Msg {
				return common.UpdateRevisionsFailedMsg{
					Err:    msg.err,
					Output: msg.output,
				}
			}
		}

		m.streamer = msg.streamer
		m.offScreenRows = nil
		m.revisionToSelect = msg.selectedRevision
		m.hasMore = true
		m.requestInFlight = false

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

		cmds := []tea.Cmd{m.requestMoreRows(msg.tag)}
		if warning := streamingWarningCmd(msg.output, msg.err); warning != nil {
			cmds = append(cmds, warning)
		}
		return tea.Batch(cmds...)
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

	if intent, ok := msg.(intents.Intent); ok {
		if cmd, handled := m.HandleIntent(intent); handled {
			return cmd
		}
		return m.op.Update(msg)
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

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if op, ok := m.op.(common.Focusable); ok && op.IsFocused() {
			return m.op.Update(keyMsg)
		}
	}

	return nil
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	// Cancel has special handling: delegate to op first, then clear selections
	if _, ok := intent.(intents.Cancel); ok {
		if h, ok := m.op.(routing.LayerHandler); ok {
			if cmd, handled := h.HandleIntent(intent); handled {
				return cmd, true
			}
		}
		// Clear checked items and reset to default operation
		if len(m.context.CheckedItems) > 0 || !m.InNormalMode() {
			m.context.ClearCheckedItems(reflect.TypeFor[appContext.SelectedRevision]())
			m.op = operations.NewDefault()
			return nil, true
		}
		return nil, false // nothing to cancel, leak to ui
	}

	// QuickSearch: swallow when an operation is active to preserve current behavior
	if _, ok := intent.(intents.QuickSearch); ok {
		if !m.InNormalMode() {
			return nil, true
		}
		return nil, false
	}

	// StartAceJump is a revisions-level concern (replaces current operation).
	// Handle before the operation to avoid delegation loops.
	if _, ok := intent.(intents.StartAceJump); ok {
		parentOp := m.op
		op := ace_jump.NewOperation(m.SetCursor, func(index int) parser.Row {
			return m.rows[index]
		}, m.displayContextRenderer.GetFirstRowIndex(), m.displayContextRenderer.GetLastRowIndex(), parentOp)
		m.op = op
		return op.Init(), true
	}

	// Try the current operation first
	if h, ok := m.op.(routing.LayerHandler); ok {
		if cmd, handled := h.HandleIntent(intent); handled {
			return cmd, true
		}
	}

	// Then try revisions-level intents
	switch intent := intent.(type) {
	case intents.OpenDetails:
		return m.openDetails(intent), true
	case intents.OpenSquash:
		return m.startSquash(intent), true
	case intents.OpenInlineDescribe:
		return m.startInlineDescribe(intent), true
	case intents.Absorb:
		return m.startAbsorb(intent), true
	case intents.OpenAbandon:
		return m.startAbandon(intent), true
	case intents.StartNew:
		return m.startNew(intent), true
	case intents.CommitWorkingCopy:
		return m.commitWorkingCopy(), true
	case intents.StartEdit:
		return m.startEdit(intent), true
	case intents.DiffEdit:
		return m.startDiffEdit(intent), true
	case intents.OpenRevert:
		return m.startRevert(intent), true
	case intents.OpenDuplicate:
		return m.startDuplicate(intent), true
	case intents.OpenSetParents:
		return m.startSetParents(intent), true
	case intents.OpenSetBookmark:
		return m.startBookmarkSet(), true
	case intents.RevisionsToggleSelect:
		commit := m.rows[m.cursor].Commit
		changeId := commit.GetChangeId()
		item := appContext.SelectedRevision{ChangeId: changeId, CommitId: commit.CommitId}
		m.context.ToggleCheckedItem(item)
		return nil, true
	case intents.Navigate:
		return m.navigate(intent), true
	case intents.Describe:
		return m.startDescribe(intent), true
	case intents.OpenEvolog:
		return m.startEvolog(intent), true
	case intents.ShowDiff:
		return m.showDiff(intent), true
	case intents.StartSplit:
		return m.startSplit(intent), true
	case intents.OpenRebase:
		return m.startRebase(intent), true
	case intents.Refresh:
		return m.refresh(intent), true
	case intents.QuickSearchCycle:
		offset := 1
		if intent.Reverse {
			offset = -1
		}
		m.SetCursor(m.search(m.cursor+offset, intent.Reverse))
		return m.updateSelection(), true
	case intents.RevisionsQuickSearchClear:
		m.quickSearch = ""
		return nil, true
	}
	return nil, false
}

func (m *Model) startBookmarkSet() tea.Cmd {
	rev := m.SelectedRevision()
	if rev == nil {
		return nil
	}
	m.op = bookmark.NewSetBookmarkOperation(m.context, rev.GetChangeId())
	return m.op.Init()
}

func (m *Model) activateOperation(op operations.Operation) tea.Cmd {
	m.op = op

	var cmds []tea.Cmd
	cmds = append(cmds, m.op.Init())
	if cur := m.SelectedRevision(); cur != nil {
		if tracked, ok := m.op.(operations.TracksSelectedRevision); ok {
			cmds = append(cmds, tracked.SetSelectedRevision(cur))
		}
	}
	return tea.Batch(cmds...)
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
	return m.activateOperation(model)
}

func (m *Model) startSquash(intent intents.OpenSquash) tea.Cmd {
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
	cmd := m.activateOperation(squash.NewOperation(m.context, selected, squash.WithFiles(intent.Files)))
	return tea.Batch(cmd, m.updateSelection())
}

func (m *Model) startRebase(intent intents.OpenRebase) tea.Cmd {
	selected := intent.Selected
	if len(selected.Revisions) == 0 {
		selected = m.SelectedRevisions()
	}
	if len(selected.Revisions) == 0 {
		return nil
	}

	source := rebaseSourceFromIntent(intent.Source)
	return m.activateOperation(rebase.NewOperation(m.context, selected, source, intent.Target))
}

func (m *Model) startRevert(intent intents.OpenRevert) tea.Cmd {
	selected := intent.Selected
	if len(selected.Revisions) == 0 {
		selected = m.SelectedRevisions()
	}
	if len(selected.Revisions) == 0 {
		return nil
	}

	return m.activateOperation(revert.NewOperation(m.context, selected, intent.Target))
}

func rebaseSourceFromIntent(source intents.RebaseSource) rebase.Source {
	switch source {
	case intents.RebaseSourceBranch:
		return rebase.SourceBranch
	case intents.RebaseSourceDescendants:
		return rebase.SourceDescendants
	default:
		return rebase.SourceRevision
	}
}

func (m *Model) startDuplicate(intent intents.OpenDuplicate) tea.Cmd {
	selected := intent.Selected
	if len(selected.Revisions) == 0 {
		selected = m.SelectedRevisions()
	}
	if len(selected.Revisions) == 0 {
		return nil
	}

	return m.activateOperation(duplicate.NewOperation(m.context, selected, intents.ModeTargetDestination))
}

func (m *Model) startSetParents(intent intents.OpenSetParents) tea.Cmd {
	commit := intent.Selected
	if commit == nil {
		commit = m.SelectedRevision()
	}
	if commit == nil {
		return nil
	}

	return m.activateOperation(set_parents.NewModel(m.context, commit))
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

func (m *Model) startDiffEdit(intent intents.DiffEdit) tea.Cmd {
	commit := intent.Selected
	if commit == nil {
		commit = m.SelectedRevision()
	}
	if commit == nil {
		return nil
	}
	return m.context.RunInteractiveCommand(jj.DiffEdit(commit.GetChangeId()), common.Refresh)
}

func (m *Model) startAbsorb(intent intents.Absorb) tea.Cmd {
	commit := intent.Selected
	if commit == nil {
		commit = m.SelectedRevision()
	}
	if commit == nil {
		return nil
	}
	return m.context.RunCommand(jj.Absorb(commit.GetChangeId()), common.Refresh)
}

func (m *Model) startAbandon(intent intents.OpenAbandon) tea.Cmd {
	selected := intent.Selected
	if len(selected.Revisions) == 0 {
		selected = m.SelectedRevisions()
	}
	if len(selected.Revisions) == 0 {
		return nil
	}
	return m.activateOperation(abandon.NewOperation(m.context, selected))
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

func (m *Model) startDescribe(intent intents.Describe) tea.Cmd {
	selected := intent.Selected
	if len(selected.Revisions) == 0 {
		selected = m.SelectedRevisions()
	}
	if len(selected.Revisions) == 0 {
		return nil
	}
	return m.context.RunInteractiveCommand(jj.Describe(selected), common.Refresh)
}

func (m *Model) startEvolog(intent intents.OpenEvolog) tea.Cmd {
	commit := intent.Selected
	if commit == nil {
		commit = m.SelectedRevision()
	}
	if commit == nil {
		return nil
	}
	model := evolog.NewOperation(m.context, commit)
	return m.activateOperation(model)
}

func (m *Model) startInlineDescribe(intent intents.OpenInlineDescribe) tea.Cmd {
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
		return intents.DiffShow{Content: string(output)}
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
	return m.context.RunInteractiveCommand(jj.Split(commit.GetChangeId(), intent.Files, intent.IsParallel, intent.IsInteractive), common.Refresh)
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

	changes := strings.SplitSeq(m.output, "\n")
	for change := range changes {
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
	if len(m.rows) == 0 {
		content := ""
		if m.isLoading {
			content = lipgloss.Place(box.R.Dx(), box.R.Dy(), lipgloss.Center, lipgloss.Center, "loading")
		} else {
			content = lipgloss.Place(box.R.Dx(), box.R.Dy(), lipgloss.Center, lipgloss.Center, "(no matching revisions)")
		}
		dl.AddDraw(box.R, content, 0)
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
		box,
		op,
		m.quickSearch,
		m.ensureCursorView,
	)

	switch overlayOp := m.op.(type) {
	case *rebase.Operation:
		overlayOp.ViewRect(dl, box)
	case *duplicate.Operation:
		overlayOp.ViewRect(dl, box)
	case *revert.Operation:
		overlayOp.ViewRect(dl, box)
	case *squash.Operation:
		overlayOp.ViewRect(dl, box)
	}

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
	return func() tea.Msg {
		if m.tag.Load() != tag {
			return nil
		}

		streamer, err := graph.NewGraphStreamer(context.Background(), m.context, revset, m.context.JJConfig.Templates.Log)
		var errMsg string
		if err != nil {
			if err == io.EOF {
				errMsg = fmt.Sprintf("No revisions found for revset `%s`", revset)
				err = errors.New(errMsg)
			} else {
				errMsg = fmt.Sprintf("%v", err)
			}
		}

		return streamingReadyMsg{
			streamer:         streamer,
			selectedRevision: selectedRevision,
			tag:              tag,
			err:              err,
			output:           errMsg,
		}
	}
}

func (m *Model) requestMoreRows(tag uint64) tea.Cmd {
	if m.requestInFlight || m.streamer == nil || !m.hasMore || tag != m.tag.Load() {
		return nil
	}

	m.requestInFlight = true
	streamer := m.streamer
	return func() tea.Msg {
		batch := streamer.RequestMore()
		return appendRowsBatchMsg{batch.Rows, batch.HasMore, tag}
	}
}

func streamingWarningCmd(output string, err error) tea.Cmd {
	if err == nil {
		return nil
	}
	if output == "" {
		output = err.Error()
	}
	return intents.Invoke(intents.AddMessage{Text: output, Err: err})
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

func (m *Model) search(startIndex int, backward bool) int {
	items := make([]screen.Searchable, len(m.rows))
	for i := range m.rows {
		items[i] = &m.rows[i]
	}
	return common.CircularSearch(items, m.quickSearch, startIndex, m.cursor, backward)
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
	m := Model{
		context:       c,
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

func (m *Model) rangeSelect(to int) {
	lo := min(m.cursor, to)
	hi := max(m.cursor, to)
	for i := lo; i <= hi; i++ {
		if i >= 0 && i < len(m.rows) {
			if commit := m.rows[i].Commit; commit != nil {
				item := appContext.SelectedRevision{ChangeId: commit.GetChangeId(), CommitId: commit.CommitId}
				m.context.ToggleCheckedItem(item)
			}
		}
	}
	m.SetCursor(to)
}

func (m *Model) jumpToParent(revisions jj.SelectedRevisions) {
	immediate, _ := m.context.RunCommandImmediate(jj.GetParent(revisions))
	parentIndex := m.selectRevision(string(immediate))
	if parentIndex != -1 {
		m.SetCursor(parentIndex)
	}
}
