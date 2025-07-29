package evolog

import (
	"bytes"

	"github.com/idursun/jjui/internal/parser"

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

type updateEvologMsg struct {
	rows []parser.Row
}

type Operation struct {
	context           *context.MainContext
	w                 *graph.Renderer
	revision          *jj.Commit
	selectedRevisions map[string]bool
	rows              []parser.Row
	cursor            int
	width             int
	height            int
	keyMap            config.KeyMappings[key.Binding]
}

func (o *Operation) restoreFromInto() (string, string) {
	from := ""
	switch f := o.context.SelectedItem.(type) {
	case context.SelectedRevision:
		from = f.ChangeId
	}
	count := 0
	into := ""
	for k, v := range o.selectedRevisions {
		if v {
			count++
			into = k
		}
	}
	if count == 1 {
		return from, into
	} else {
		return "", ""
	}
}

func (o *Operation) ShortHelp() []key.Binding {
	binds := []key.Binding{o.keyMap.Up, o.keyMap.Down, o.keyMap.Cancel, o.keyMap.Diff}

	if from, into := o.restoreFromInto(); from != "" && into != "" {
		binds = append(binds, key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "restore descendants from "+from+" into "+into),
		))
	}

	return binds
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
		case msg.String() == "R":
			if from, into := o.restoreFromInto(); from != "" && into != "" {
				args := []string{"restore", "--from", from, "--into", into, "--restore-descendants"}
				return o, o.context.RunCommand(jj.Args(args...), common.Close, common.Refresh, common.Close)
			}
		}
	}
	return o, o.updateSelection()
}

func (o *Operation) updateSelection() tea.Cmd {
	if o.rows == nil {
		return nil
	}

	return o.context.SetSelectedItem(context.SelectedRevision{
		ChangeId: o.rows[o.cursor].Commit.GetChangeId(),
		CommitId: o.rows[o.cursor].Commit.CommitId,
	})
}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
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
	return "evolog"
}

func (o *Operation) load() tea.Msg {
	output, _ := o.context.RunCommandImmediate(jj.Evolog(o.revision.GetChangeId()))
	rows := parser.ParseRows(bytes.NewReader(output))
	return updateEvologMsg{
		rows: rows,
	}
}

func NewOperation(context *context.MainContext, revision *jj.Commit, selectedRevisions map[string]bool, width int, height int) (operations.Operation, tea.Cmd) {
	w := graph.NewRenderer(width, height)
	o := &Operation{
		context:           context,
		keyMap:            config.Current.GetKeyMap(),
		w:                 w,
		revision:          revision,
		selectedRevisions: selectedRevisions,
		rows:              nil,
		cursor:            0,
		width:             width,
		height:            height,
	}
	return o, o.load
}
