package evolog

import (
	"bytes"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/graph"
	"github.com/idursun/jjui/internal/ui/operations"
)

type viewRange struct {
	start int
	end   int
}

type updateEvologMsg struct {
	rows []graph.Row
}

type Operation struct {
	context   context.AppContext
	revision  string
	rows      []graph.Row
	viewRange *viewRange
	cursor    int
	width     int
	height    int
	keyMap    config.KeyMappings[key.Binding]
}

func (o Operation) ShortHelp() []key.Binding {
	return []key.Binding{o.keyMap.Up, o.keyMap.Down, o.keyMap.Cancel, o.keyMap.Diff}
}

func (o Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{o.ShortHelp()}
}

func (o Operation) Update(msg tea.Msg) (operations.OperationWithOverlay, tea.Cmd) {
	switch msg := msg.(type) {
	case updateEvologMsg:
		o.rows = msg.rows
		o.cursor = 0
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, o.keyMap.Cancel):
			return o, common.Close
		case key.Matches(msg, o.keyMap.Diff):
			return o, func() tea.Msg {
				selectedCommitId := o.rows[o.cursor].Commit.CommitId
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
		}
	}
	return o, o.updateSelection()
}

func (o Operation) updateSelection() tea.Cmd {
	if o.rows == nil {
		return nil
	}

	return o.context.SetSelectedItem(context.SelectedRevision{ChangeId: o.rows[o.cursor].Commit.CommitId})
}

func (o Operation) RenderPosition() operations.RenderPosition {
	return operations.RenderPositionAfter
}

func (o Operation) Render() string {
	if len(o.rows) == 0 {
		return "loading"
	}
	h := min(o.height-5, len(o.rows)*2)
	highlightBackground := lipgloss.AdaptiveColor{
		Light: config.Current.UI.HighlightLight,
		Dark:  config.Current.UI.HighlightDark,
	}
	var w graph.Renderer
	selectedLineStart := -1
	selectedLineEnd := -1
	for i, row := range o.rows {
		nodeRenderer := graph.DefaultRowDecorator{
			Palette:             common.DefaultPalette,
			HighlightBackground: highlightBackground,
			Op:                  &operations.Default{},
			IsHighlighted:       i == o.cursor,
			Width:               o.width,
		}

		if i == o.cursor {
			selectedLineStart = w.LineCount()
		}
		graph.RenderRow(&w, row, nodeRenderer)
		if i == o.cursor {
			selectedLineEnd = w.LineCount()
		}
		if selectedLineEnd > 0 && w.LineCount() > h && w.LineCount() > o.viewRange.end {
			break
		}
	}

	if selectedLineStart <= o.viewRange.start {
		o.viewRange.start = selectedLineStart
		o.viewRange.end = selectedLineStart + h
	} else if selectedLineEnd > o.viewRange.end {
		o.viewRange.end = selectedLineEnd
		o.viewRange.start = selectedLineEnd - h
	}

	content := w.String(o.viewRange.start, o.viewRange.end)
	content = lipgloss.PlaceHorizontal(o.width, lipgloss.Left, content)
	return content
}

func (o Operation) Name() string {
	return "evolog"
}

func (o Operation) load() tea.Msg {
	output, _ := o.context.RunCommandImmediate(jj.Evolog(o.revision))
	rows := graph.ParseRows(bytes.NewReader(output))
	return updateEvologMsg{
		rows: rows,
	}
}

func NewOperation(context context.AppContext, revision string, width int, height int) (*Operation, tea.Cmd) {
	v := viewRange{start: 0, end: 0}
	o := Operation{
		context:   context,
		keyMap:    context.KeyMap(),
		revision:  revision,
		rows:      nil,
		viewRange: &v,
		cursor:    0,
		width:     width,
		height:    height,
	}
	return &o, o.load
}
