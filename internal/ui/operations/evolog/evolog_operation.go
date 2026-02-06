package evolog

import (
	"bytes"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/context"
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
var _ common.Focusable = (*Operation)(nil)
var _ common.Overlay = (*Operation)(nil)

type Operation struct {
	context          *context.MainContext
	dlRenderer       *render.ListRenderer
	revision         *jj.Commit
	mode             mode
	rows             []parser.Row
	cursor           int
	keyMap           config.KeyMappings[key.Binding]
	target           *jj.Commit
	styles           styles
	ensureCursorView bool
}

func (o *Operation) IsOverlay() bool {
	return o.mode == selectMode
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

func (o *Operation) Len() int {
	return len(o.rows)
}

func (o *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, o.keyMap.Cancel):
		return o.handleIntent(intents.Cancel{})
	case key.Matches(msg, o.keyMap.Quit):
		return o.handleIntent(intents.Quit{})
	case key.Matches(msg, o.keyMap.Up):
		return o.handleIntent(intents.EvologNavigate{Delta: -1})
	case key.Matches(msg, o.keyMap.Down):
		return o.handleIntent(intents.EvologNavigate{Delta: 1})
	case key.Matches(msg, o.keyMap.Evolog.Diff):
		return o.handleIntent(intents.EvologDiff{})
	case key.Matches(msg, o.keyMap.Evolog.Restore):
		return o.handleIntent(intents.EvologRestore{})
	case key.Matches(msg, o.keyMap.Apply):
		return o.handleIntent(intents.Apply{})
	}
	return nil
}

type styles struct {
	dimmedStyle   lipgloss.Style
	commitIdStyle lipgloss.Style
	changeIdStyle lipgloss.Style
	markerStyle   lipgloss.Style
	textStyle     lipgloss.Style
	selectedStyle lipgloss.Style
}

func (o *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	o.target = commit
	return nil
}

func (o *Operation) ShortHelp() []key.Binding {
	if o.mode == restoreMode {
		return []key.Binding{o.keyMap.Cancel, o.keyMap.Apply}
	}
	return []key.Binding{
		o.keyMap.Up,
		o.keyMap.Down,
		o.keyMap.Cancel,
		o.keyMap.Quit,
		o.keyMap.Evolog.Diff,
		o.keyMap.Evolog.Restore,
	}
}

func (o *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
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
		return o.handleIntent(msg)
	case tea.KeyMsg:
		cmd := o.HandleKey(msg)
		return cmd
	}
	return nil
}

func (o *Operation) handleIntent(intent intents.Intent) tea.Cmd {
	switch msg := intent.(type) {
	case intents.Quit:
		return tea.Quit
	case intents.Cancel:
		if o.mode == restoreMode {
			o.mode = selectMode
			return nil
		}
		return common.Close
	case intents.EvologNavigate:
		if o.mode != selectMode {
			return nil
		}
		if msg.Delta < 0 && o.cursor > 0 {
			o.cursor--
			o.ensureCursorView = true
			return o.updateSelection()
		}
		if msg.Delta > 0 && o.cursor < len(o.rows)-1 {
			o.cursor++
			o.ensureCursorView = true
			return o.updateSelection()
		}
		return nil
	case intents.EvologDiff:
		if o.mode != selectMode {
			return nil
		}
		return func() tea.Msg {
			selectedCommitId := o.getSelectedEvolog().CommitId
			output, _ := o.context.RunCommandImmediate(jj.Diff(selectedCommitId, ""))
			return common.ShowDiffMsg(output)
		}
	case intents.EvologRestore:
		if o.mode != selectMode {
			return nil
		}
		o.mode = restoreMode
		return nil
	case intents.Apply:
		if o.mode != restoreMode {
			return nil
		}
		from := o.getSelectedEvolog().CommitId
		into := o.target.GetChangeId()
		return o.context.RunCommand(jj.RestoreEvolog(from, into), common.Close, common.Refresh)
	}
	return nil
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
		selectedCommitId := o.getSelectedEvolog().CommitId
		return lipgloss.JoinHorizontal(0,
			o.styles.markerStyle.Render("<< restore >>"),
			o.styles.dimmedStyle.PaddingLeft(1).Render("restore from "),
			o.styles.commitIdStyle.Render(selectedCommitId),
			o.styles.dimmedStyle.Render(" into "),
			o.styles.changeIdStyle.Render(o.target.GetChangeId()),
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
	// In selectMode with isSelected and pos==After, RenderToDisplayContext handles rendering
	return ""
}

func (o *Operation) Name() string {
	if o.mode == restoreMode {
		return "restore"
	}
	return "evolog"
}

// DesiredHeight returns the desired height for the operation
func (o *Operation) DesiredHeight(commit *jj.Commit, pos operations.RenderPosition) int {
	isSelected := commit.GetChangeId() == o.revision.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter || o.mode != selectMode {
		return 0
	}
	if len(o.rows) == 0 {
		return 1 // "loading" message
	}
	// Sum up all row heights
	total := 0
	for _, row := range o.rows {
		total += len(row.Lines)
	}
	return total
}

// RenderToDisplayContext renders the evolog list directly to the DisplayContext
func (o *Operation) RenderToDisplayContext(dl *render.DisplayContext, commit *jj.Commit, pos operations.RenderPosition, rect cellbuf.Rectangle, _ cellbuf.Position) int {
	isSelected := commit.GetChangeId() == o.revision.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter || o.mode != selectMode {
		return 0
	}

	return o.renderListToDisplayContext(dl, rect, o.ensureCursorView)
}

func (o *Operation) load() tea.Msg {
	output, _ := o.context.RunCommandImmediate(jj.Evolog(o.revision.GetChangeId()))
	rows := parser.ParseRows(bytes.NewReader(output))
	return updateEvologMsg{
		rows: rows,
	}
}

func NewOperation(context *context.MainContext, revision *jj.Commit) *Operation {
	styles := styles{
		dimmedStyle:   common.DefaultPalette.Get("evolog dimmed"),
		commitIdStyle: common.DefaultPalette.Get("evolog commit_id"),
		changeIdStyle: common.DefaultPalette.Get("evolog change_id"),
		markerStyle:   common.DefaultPalette.Get("evolog target_marker"),
		textStyle:     common.DefaultPalette.Get("evolog text"),
		selectedStyle: common.DefaultPalette.Get("evolog selected"),
	}
	o := &Operation{
		context:    context,
		keyMap:     config.Current.GetKeyMap(),
		revision:   revision,
		rows:       nil,
		cursor:     0,
		styles:     styles,
		dlRenderer: render.NewListRenderer(EvologScrollMsg{}),
	}
	return o
}

func (o *Operation) renderListToDisplayContext(
	dl *render.DisplayContext,
	rect cellbuf.Rectangle,
	ensureCursorVisible bool,
) int {
	if len(o.rows) == 0 {
		content := "loading"
		dl.AddDraw(cellbuf.Rect(rect.Min.X, rect.Min.Y, rect.Dx(), 1), content, 0)
		return 1
	}

	totalHeight := 0
	for _, row := range o.rows {
		totalHeight += len(row.Lines)
	}
	height := min(rect.Dy(), totalHeight)

	measure := func(index int) int {
		return len(o.rows[index].Lines)
	}

	renderItem := func(dl *render.DisplayContext, index int, itemRect cellbuf.Rectangle) {
		row := o.rows[index]
		isItemSelected := index == o.cursor
		styleOverride := o.styles.textStyle
		if isItemSelected {
			styleOverride = o.styles.selectedStyle
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
			lineContent := lipgloss.PlaceHorizontal(itemRect.Dx(), 0, content.String(), lipgloss.WithWhitespaceBackground(styleOverride.GetBackground()))
			lineRect := cellbuf.Rect(itemRect.Min.X, y, itemRect.Dx(), 1)
			dl.AddDraw(lineRect, lineContent, 0)

			if isItemSelected {
				dl.AddHighlight(lineRect, o.styles.selectedStyle, 1)
			}
			y++
		}
	}

	clickMsg := func(index int) render.ClickMessage {
		return EvologClickedMsg{Index: index}
	}

	viewRect := layout.Box{R: cellbuf.Rect(rect.Min.X, rect.Min.Y, rect.Dx(), height)}
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
