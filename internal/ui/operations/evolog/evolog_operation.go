package evolog

import (
	"bytes"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/operations"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/graph"
)

type updateEvologMsg struct {
	rows []parser.Row
}

type mode int

const (
	selectMode mode = iota
	restoreMode
)

type Operation struct {
	context  *context.MainContext
	w        *graph.Renderer
	revision *jj.Commit
	mode     mode
	rows     []parser.Row
	cursor   int
	width    int
	height   int
	keyMap   config.KeyMappings[key.Binding]
	target   *jj.Commit
	styles   styles
}

func (o *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch o.mode {
	case selectMode:
		switch {
		case key.Matches(msg, o.keyMap.Cancel):
			return common.Close
		case key.Matches(msg, o.keyMap.Diff):
			return func() tea.Msg {
				selectedCommitId := o.getSelectedEvolog().CommitId
				output, _ := o.context.RunCommandImmediate(jj.Diff(selectedCommitId, ""))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, o.keyMap.Up):
			if o.cursor > 0 {
				o.cursor--
			}
		case key.Matches(msg, o.keyMap.Down):
			if o.cursor < len(o.rows)-1 {
				o.cursor++
			}
		case key.Matches(msg, restoreKey):
			o.mode = restoreMode
		}
	case restoreMode:
		switch {
		case key.Matches(msg, o.keyMap.Cancel):
			o.mode = selectMode
			return nil
		case key.Matches(msg, o.keyMap.Apply):
			from := o.getSelectedEvolog().CommitId
			into := o.target.GetChangeId()
			return o.context.RunCommand(jj.RestoreEvolog(from, into), common.Close, common.Refresh)
		}
	}
	return nil
}

type styles struct {
	dimmedStyle   lipgloss.Style
	commitIdStyle lipgloss.Style
	changeIdStyle lipgloss.Style
	markerStyle   lipgloss.Style
}

func (o *Operation) SetSelectedRevision(commit *jj.Commit) {
	o.target = commit
}

// TODO: move this to the default keymap
var restoreKey = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restore"))

func (o *Operation) ShortHelp() []key.Binding {
	if o.mode == restoreMode {
		return []key.Binding{o.keyMap.Cancel, o.keyMap.Apply}
	}
	return []key.Binding{o.keyMap.Up, o.keyMap.Down, o.keyMap.Cancel, o.keyMap.Diff, restoreKey}
}

func (o *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
}

func (o *Operation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	switch msg := msg.(type) {
	case updateEvologMsg:
		o.rows = msg.rows
		o.cursor = 0
	case tea.KeyMsg:
		cmd := o.HandleKey(msg)
		return o, cmd
	}
	return o, o.updateSelection()
}

func (o *Operation) getSelectedEvolog() *jj.Commit {
	return o.rows[o.cursor].Commit
}

func (o *Operation) updateSelection() tea.Cmd {
	if o.rows == nil {
		return nil
	}

	selected := o.getSelectedEvolog()
	return o.context.SetSelectedItem(context.SelectedRevision{
		ChangeId: selected.GetChangeId(),
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

	if len(o.rows) == 0 {
		return "loading"
	}
	h := min(o.height-5, len(o.rows)*2)
	o.w.SetSize(o.width, h)
	renderer := graph.NewDefaultRowIterator(o.rows, graph.WithWidth(o.width), graph.WithStylePrefix("evolog"))
	renderer.Cursor = o.cursor
	content := o.w.Render(renderer)
	content = lipgloss.PlaceHorizontal(o.width, lipgloss.Left, content)
	return content
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

func NewOperation(context *context.MainContext, revision *jj.Commit, width int, height int) (operations.Operation, tea.Cmd) {
	styles := styles{
		dimmedStyle:   common.DefaultPalette.Get("evolog dimmed"),
		commitIdStyle: common.DefaultPalette.Get("evolog commit_id"),
		changeIdStyle: common.DefaultPalette.Get("evolog change_id"),
		markerStyle:   common.DefaultPalette.Get("evolog target_marker"),
	}
	w := graph.NewRenderer(width, height)
	o := &Operation{
		context:  context,
		keyMap:   config.Current.GetKeyMap(),
		w:        w,
		revision: revision,
		rows:     nil,
		cursor:   0,
		width:    width,
		height:   height,
		styles:   styles,
	}
	return o, o.load
}
