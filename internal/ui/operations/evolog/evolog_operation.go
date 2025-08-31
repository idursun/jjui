package evolog

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context/models"
	"github.com/idursun/jjui/internal/ui/operations"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/graph"
)

type mode int

const (
	selectMode mode = iota
	restoreMode
)

type Operation struct {
	revisionsContext *context.RevisionsContext
	context          *context.EvologContext
	w                *graph.Renderer
	revision         *models.RevisionItem
	mode             mode
	width            int
	height           int
	keyMap           config.KeyMappings[key.Binding]
	styles           styles
}

func (o *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch o.mode {
	case selectMode:
		switch {
		case key.Matches(msg, o.keyMap.Cancel):
			return common.Close
		case key.Matches(msg, o.keyMap.Up):
			o.context.Prev()
		case key.Matches(msg, o.keyMap.Down):
			o.context.Next()
		case key.Matches(msg, o.keyMap.Evolog.Diff):
			return func() tea.Msg {
				selectedCommitId := o.context.Current().Commit.CommitId
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
			from := o.context.Current().Commit.CommitId
			into := o.revisionsContext.Current().Commit.GetChangeId()
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
	case tea.KeyMsg:
		cmd := o.HandleKey(msg)
		return o, cmd
	}
	return o, nil
}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	target := o.revisionsContext.Current().Commit
	if o.mode == restoreMode && pos == operations.RenderPositionBefore && target != nil && target.GetChangeId() == commit.GetChangeId() {
		selectedCommitId := o.context.Current().Commit.CommitId
		return lipgloss.JoinHorizontal(0,
			o.styles.markerStyle.Render("<< restore >>"),
			o.styles.dimmedStyle.PaddingLeft(1).Render("restore from "),
			o.styles.commitIdStyle.Render(selectedCommitId),
			o.styles.dimmedStyle.Render(" into "),
			o.styles.changeIdStyle.Render(target.GetChangeId()),
		)
	}

	// if we are in restore mode, we don't render evolog list
	if o.mode == restoreMode {
		return ""
	}

	isSelected := commit.GetChangeId() == o.revision.Commit.GetChangeId()
	if !isSelected || pos != operations.RenderPositionAfter {
		return ""
	}

	if len(o.context.Items) == 0 {
		return "loading"
	}
	h := min(o.height-5, len(o.context.Items)*2)
	o.w.SetSize(o.width, h)
	rows := make([]parser.Row, 0)
	for _, item := range o.context.Items {
		rows = append(rows, *item.Row)
	}
	renderer := graph.NewDefaultRowIterator(rows, graph.WithWidth(o.width), graph.WithStylePrefix("evolog"))
	renderer.Cursor = o.context.Cursor()
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

func NewOperation(context *context.RevisionsContext, width int, height int) (operations.Operation, tea.Cmd) {
	current := context.Current()
	context.EvologContext.Load(current)
	styles := styles{
		dimmedStyle:   common.DefaultPalette.Get("evolog dimmed"),
		commitIdStyle: common.DefaultPalette.Get("evolog commit_id"),
		changeIdStyle: common.DefaultPalette.Get("evolog change_id"),
		markerStyle:   common.DefaultPalette.Get("evolog target_marker"),
	}
	w := graph.NewRenderer(width, height)
	o := &Operation{
		revisionsContext: context,
		context:          context.EvologContext,
		keyMap:           config.Current.GetKeyMap(),
		w:                w,
		revision:         current,
		width:            width,
		height:           height,
		styles:           styles,
	}
	return o, nil
}
