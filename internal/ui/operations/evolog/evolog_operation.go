package evolog

import (
	"bytes"
	"fmt"
	"io"
	"strings"

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
)

type updateEvologMsg struct {
	rows []*models.RevisionItem
}

type mode int

const (
	selectMode mode = iota
	restoreMode
)

type EvologList struct {
	*list.List[*models.RevisionItem]
	renderer      *list.ListRenderer[*models.RevisionItem]
	selectedStyle lipgloss.Style
	textStyle     lipgloss.Style
	dimmedStyle   lipgloss.Style
	commitIdStyle lipgloss.Style
	changeIdStyle lipgloss.Style
	markerStyle   lipgloss.Style
}

func (e *EvologList) RenderItem(w io.Writer, index int) {
	row := e.Items[index]
	isHighlighted := index == e.Cursor
	for lineIndex := 0; lineIndex < len(row.Lines); lineIndex++ {
		segmentedLine := row.Lines[lineIndex]

		lw := strings.Builder{}
		for _, segment := range segmentedLine.Gutter.Segments {
			style := segment.Style
			fmt.Fprint(&lw, style.Render(segment.Text))
		}

		for _, segment := range segmentedLine.Segments {
			style := segment.Style
			if isHighlighted {
				style = style.Inherit(e.selectedStyle)
			}
			fmt.Fprint(&lw, style.Render(segment.Text))
		}
		line := lw.String()
		if isHighlighted && segmentedLine.Flags&models.Highlightable == models.Highlightable {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(e.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(e.selectedStyle.GetBackground())))
		} else {
			fmt.Fprint(w, lipgloss.PlaceHorizontal(e.renderer.Width, 0, line, lipgloss.WithWhitespaceBackground(e.textStyle.GetBackground())))
		}
		fmt.Fprint(w, "\n")
	}
}

func (e *EvologList) GetItemHeight(index int) int {
	return len(e.Items[index].Lines)
}

type Operation struct {
	*common.Sizeable
	*EvologList
	context  *context.MainContext
	revision *jj.Commit
	mode     mode
	keyMap   config.KeyMappings[key.Binding]
	target   *jj.Commit
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
			o.markerStyle.Render("<< restore >>"),
			o.dimmedStyle.PaddingLeft(1).Render("restore from "),
			o.commitIdStyle.Render(selectedCommitId),
			o.dimmedStyle.Render(" into "),
			o.changeIdStyle.Render(o.target.GetChangeId()),
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
	o.renderer.SetHeight(h)
	content := o.renderer.Render()
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
	size := common.NewSizeable(width, height)
	l := list.NewList[*models.RevisionItem]()
	el := &EvologList{
		List:          l,
		selectedStyle: common.DefaultPalette.Get("evolog selected"),
		textStyle:     common.DefaultPalette.Get("evolog text"),
		dimmedStyle:   common.DefaultPalette.Get("evolog dimmed"),
		commitIdStyle: common.DefaultPalette.Get("evolog commit_id"),
		changeIdStyle: common.DefaultPalette.Get("evolog change_id"),
		markerStyle:   common.DefaultPalette.Get("evolog target_marker"),
	}
	el.renderer = list.NewRenderer[*models.RevisionItem](l, el.RenderItem, el.GetItemHeight, common.NewSizeable(width, height))
	o := &Operation{
		Sizeable:   size,
		EvologList: el,
		context:    context,
		keyMap:     config.Current.GetKeyMap(),
		revision:   revision,
	}
	return o, o.load
}
