package evolog

import (
	"bytes"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/common/models"
	"github.com/idursun/jjui/internal/ui/operations"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/graph"
)

type updateEvologMsg struct {
	rows []*models.RevisionItem
}

type mode int

const (
	selectMode mode = iota
	restoreMode
)

type Operation struct {
	*common.Sizeable
	*list.List[*models.RevisionItem]
	context  *context.MainContext
	w        *graph.Renderer
	revision *jj.Commit
	mode     mode
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
		case key.Matches(msg, o.keyMap.Up):
			if o.Cursor > 0 {
				o.Cursor--
				return o.updateSelection()
			}
		case key.Matches(msg, o.keyMap.Down):
			if o.Cursor < len(o.Items)-1 {
				o.Cursor++
				return o.updateSelection()
			}
		case key.Matches(msg, o.keyMap.Evolog.Diff):
			return func() tea.Msg {
				selectedCommitId := o.getSelectedEvolog().CommitId
				output, _ := o.context.RunCommandImmediate(jj.Diff(selectedCommitId, ""))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, o.keyMap.Evolog.Restore):
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

func (o *Operation) ShortHelp() []key.Binding {
	if o.mode == restoreMode {
		return []key.Binding{o.keyMap.Cancel, o.keyMap.Apply}
	}
	return []key.Binding{o.keyMap.Up, o.keyMap.Down, o.keyMap.Cancel, o.keyMap.Evolog.Diff, o.keyMap.Evolog.Restore}
}

func (o *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
}

func (o *Operation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	switch msg := msg.(type) {
	case updateEvologMsg:
		o.Items = msg.rows
		o.Cursor = 0
		return o, o.updateSelection()
	case tea.KeyMsg:
		cmd := o.HandleKey(msg)
		return o, cmd
	}
	return o, nil
}

func (o *Operation) getSelectedEvolog() *jj.Commit {
	return o.Items[o.Cursor].Commit
}

func (o *Operation) updateSelection() tea.Cmd {
	if o.Items == nil {
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

	if len(o.Items) == 0 {
		return "loading"
	}
	h := min(o.Height-5, len(o.Items)*2)
	o.w.SetSize(o.Width, h)
	var rows []models.Row
	for _, item := range o.Items {
		rows = append(rows, item.Row)
	}
	renderer := graph.NewDefaultRowIterator(rows, graph.WithWidth(o.Width), graph.WithStylePrefix("evolog"))
	renderer.Cursor = o.Cursor
	content := o.w.Render(renderer)
	content = lipgloss.PlaceHorizontal(o.Width, lipgloss.Left, content)
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
		Sizeable: &common.Sizeable{Width: width, Height: height},
		List:     list.NewList[*models.RevisionItem](),
		context:  context,
		keyMap:   config.Current.GetKeyMap(),
		w:        w,
		revision: revision,
		styles:   styles,
	}
	return o, o.load
}
