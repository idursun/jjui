package evolog

import (
	"bytes"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

type updateEvologMsg struct {
	rows []parser.Row
}

type EvologClickedMsg struct {
	Index int
}

type EvologScrollMsg struct {
	Delta      int
	Horizontal bool
}

func (e EvologScrollMsg) SetDelta(delta int, horizontal bool) tea.Msg {
	return EvologScrollMsg{Delta: delta, Horizontal: horizontal}
}

type mode int

const (
	selectMode mode = iota
	restoreMode
)

var _ operations.Operation = (*Operation)(nil)
var _ operations.EmbeddedOperation = (*Operation)(nil)
var _ common.Focusable = (*Operation)(nil)
var _ common.Overlay = (*Operation)(nil)
var _ dispatch.ScopeProvider = (*Operation)(nil)

type Operation struct {
	context          *context.MainContext
	dlRenderer       *render.ListRenderer
	revision         *jj.Commit
	mode             mode
	rows             []parser.Row
	cursor           int
	target           *jj.Commit
	ensureCursorView bool
}

func (o *Operation) Len() int {
	if o.rows == nil {
		return 0
	}
	return len(o.rows)
}

func (o *Operation) Cursor() int {
	return o.cursor
}

func (o *Operation) SetCursor(index int) {
	if index >= 0 && index < len(o.rows) {
		o.cursor = index
		o.ensureCursorView = true
	}
}

func (o *Operation) IsOverlay() bool {
	return o.mode == selectMode
}

func (o *Operation) Scopes() []dispatch.Scope {
	leak := dispatch.LeakGlobal
	if o.mode == restoreMode {
		leak = dispatch.LeakAll
	}
	return []dispatch.Scope{
		{
			Name:    actions.ScopeEvolog,
			Leak:    leak,
			Handler: o,
		},
	}
}

func (o *Operation) IsFocused() bool {
	return true
}

func (o *Operation) Init() tea.Cmd {
	return o.load
}

func (o *Operation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	o.renderListToDisplayContext(dl, box.R, o.ensureCursorView)
}

func (o *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	o.target = commit
	return nil
}

func (o *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case updateEvologMsg:
		o.rows = msg.rows
		o.cursor = 0
		o.ensureCursorView = true
		return o.updateSelection()
	case EvologClickedMsg:
		if msg.Index >= 0 && msg.Index < len(o.rows) {
			o.cursor = msg.Index
			o.ensureCursorView = true
			return o.updateSelection()
		}
	case EvologScrollMsg:
		if msg.Horizontal {
			return nil
		}
		o.ensureCursorView = false
		o.scroll(msg.Delta)
		return nil
	case intents.Intent:
		cmd, _ := o.HandleIntent(msg)
		return cmd
	}
	return nil
}

func (o *Operation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.Quit:
		return tea.Quit, true
	case intents.Cancel:
		return common.Close, true
	case intents.EvologNavigate:
		if o.mode == restoreMode {
			return intents.Invoke(intents.Navigate{Delta: intent.Delta}), true
		}
		return o.navigate(intent.Delta, intent.IsPage), true
	case intents.EvologDiff:
		if o.mode != selectMode {
			return nil, true
		}
		return func() tea.Msg {
			selectedCommitId := o.getSelectedEvolog().CommitId
			output, _ := o.context.RunCommandImmediate(jj.Diff(selectedCommitId, ""))
			return intents.DiffShow{Content: string(output)}
		}, true
	case intents.EvologRestore:
		if o.mode != selectMode {
			return nil, true
		}
		o.mode = restoreMode
		return nil, true
	case intents.Apply:
		if o.mode != restoreMode {
			return nil, true
		}
		from := o.getSelectedEvolog().CommitId
		into := o.target.GetChangeId()
		return o.context.RunCommand(jj.RestoreEvolog(from, into), common.CloseApplied, common.Refresh), true
	}
	return nil, false
}

func (o *Operation) navigate(delta int, page bool) tea.Cmd {
	if o.Len() == 0 {
		return nil
	}

	// Calculate step (convert page scroll to item count)
	step := delta
	if page {
		firstRowIndex := o.dlRenderer.GetFirstRowIndex()
		lastRowIndex := o.dlRenderer.GetLastRowIndex()
		span := max(lastRowIndex-firstRowIndex-1, 1)
		if step < 0 {
			step = -span
		} else {
			step = span
		}
	}

	// Calculate new cursor position
	totalItems := len(o.rows)
	newCursor := o.cursor + step
	if newCursor < 0 {
		newCursor = 0
	} else if newCursor >= totalItems {
		newCursor = totalItems - 1
	}

	o.SetCursor(newCursor)
	return o.updateSelection()
}

func (o *Operation) getSelectedEvolog() *jj.Commit {
	return o.rows[o.cursor].Commit
}

func (o *Operation) updateSelection() tea.Cmd {
	if o.rows == nil {
		return nil
	}

	selected := o.getSelectedEvolog()
	return o.context.SetSelectedItem(context.SelectedCommit{
		CommitId: selected.CommitId,
	})
}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if o.mode == restoreMode && pos == operations.RenderPositionBefore && o.target != nil && o.target.GetChangeId() == commit.GetChangeId() {

		dimmedStyle := common.DefaultPalette.Get("evolog dimmed")
		commitIdStyle := common.DefaultPalette.Get("evolog commit_id")
		changeIdStyle := common.DefaultPalette.Get("evolog change_id")
		markerStyle := common.DefaultPalette.Get("evolog target_marker")

		selectedCommitId := o.getSelectedEvolog().CommitId
		return lipgloss.JoinHorizontal(0,
			markerStyle.Render("<< restore >>"),
			dimmedStyle.PaddingLeft(1).Render("restore from "),
			commitIdStyle.Render(selectedCommitId),
			dimmedStyle.Render(" into "),
			changeIdStyle.Render(o.target.GetChangeId()),
		)
	}

	// if we are in restore mode, we don't render evolog list
	if o.mode == restoreMode {
		return ""
	}

	isSelected := commit.GetChangeId() == o.revision.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter {
		return ""
	}
	return ""
}

func (o *Operation) CanEmbed(commit *jj.Commit, pos operations.RenderPosition) bool {
	isSelected := commit.GetChangeId() == o.revision.GetChangeId()
	return isSelected && pos == operations.RenderPositionAfter && o.mode == selectMode
}

func (o *Operation) EmbeddedHeight(commit *jj.Commit, pos operations.RenderPosition, _ int) int {
	if !o.CanEmbed(commit, pos) {
		return 0
	}
	if len(o.rows) == 0 {
		return 1
	}
	total := 0
	for _, row := range o.rows {
		total += len(row.Lines)
	}
	return total
}

func (o *Operation) Name() string {
	if o.mode == restoreMode {
		return "restore"
	}
	return "evolog"
}

func (o *Operation) load() tea.Msg {
	output, _ := o.context.RunCommandImmediate(jj.Evolog(o.revision.GetChangeId()))
	rows := parser.ParseRows(bytes.NewReader(output))
	return updateEvologMsg{
		rows: rows,
	}
}

func NewOperation(context *context.MainContext, revision *jj.Commit) *Operation {
	o := &Operation{
		context:    context,
		revision:   revision,
		rows:       nil,
		cursor:     0,
		dlRenderer: render.NewListRenderer(EvologScrollMsg{}),
	}
	return o
}

func (o *Operation) renderListToDisplayContext(
	dl *render.DisplayContext,
	rect layout.Rectangle,
	ensureCursorVisible bool,
) int {
	if len(o.rows) == 0 {
		content := "loading"
		dl.AddDraw(layout.Rect(rect.Min.X, rect.Min.Y, rect.Dx(), 1), content, 0)
		return 1
	}
	textStyle := common.DefaultPalette.Get("evolog text")
	selectedStyle := common.DefaultPalette.Get("evolog selected")

	totalHeight := 0
	for _, row := range o.rows {
		totalHeight += len(row.Lines)
	}
	height := min(rect.Dy(), totalHeight)

	measure := func(index int) int {
		return len(o.rows[index].Lines)
	}

	renderItem := func(dl *render.DisplayContext, index int, itemRect layout.Rectangle) {
		row := o.rows[index]
		isItemSelected := index == o.cursor
		styleOverride := textStyle
		if isItemSelected {
			styleOverride = selectedStyle
		}

		y := itemRect.Min.Y
		for _, line := range row.Lines {
			var content strings.Builder
			for _, segment := range line.Gutter.Segments {
				content.WriteString(segment.Style.Render(segment.Text))
			}
			for _, segment := range line.Segments {
				style := segment.Style.Inherit(styleOverride)
				content.WriteString(style.Render(segment.Text))
			}
			lineContent := lipgloss.PlaceHorizontal(itemRect.Dx(), 0, content.String(), lipgloss.WithWhitespaceStyle(styleOverride))
			lineRect := layout.Rect(itemRect.Min.X, y, itemRect.Dx(), 1)
			dl.AddDraw(lineRect, lineContent, 0)

			if isItemSelected {
				dl.AddHighlight(lineRect, selectedStyle, 1)
			}
			y++
		}
	}

	clickMsg := func(index int, _ tea.Mouse) render.ClickMessage {
		return EvologClickedMsg{Index: index}
	}

	viewRect := layout.Box{R: layout.Rect(rect.Min.X, rect.Min.Y, rect.Dx(), height)}
	o.dlRenderer.Render(
		dl,
		viewRect,
		len(o.rows),
		o.cursor,
		ensureCursorVisible,
		measure,
		renderItem,
		clickMsg,
	)
	o.dlRenderer.RegisterScroll(dl, viewRect)

	return height
}

func (o *Operation) scroll(delta int) {
	currentStart := o.dlRenderer.GetScrollOffset()
	desiredStart := currentStart + delta
	o.dlRenderer.SetScrollOffset(desiredStart)
}
