package evolog

import (
	"bytes"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"

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

var (
	_ view.IViewModel      = (*Operation)(nil)
	_ operations.Operation = (*Operation)(nil)
)

type Operation struct {
	*EvologList
	*view.ViewNode
	context  *context.MainContext
	revision *models.Commit
	mode     mode
	keyMap   config.KeyMappings[key.Binding]
}

func (o *Operation) Mount(v *view.ViewNode) {
	o.ViewNode = v
	v.Height = v.ViewManager.Height
	o.renderer.Sizeable = view.NewSizeable(v.Parent.Width, v.Parent.Height)
	v.Id = o.GetId()
}

func (o *Operation) GetId() view.ViewId {
	return "evolog"
}

func (o *Operation) Init() tea.Cmd {
	o.context.Evolog.SetItems(nil)
	return o.load
}

func (o *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch o.mode {
	case selectMode:
		switch {
		case key.Matches(msg, o.keyMap.Cancel):
			o.ViewManager.UnregisterView(o.Id)
			return nil
		case key.Matches(msg, o.keyMap.Up):
			o.CursorUp()
		case key.Matches(msg, o.keyMap.Down):
			o.CursorDown()
		case key.Matches(msg, o.keyMap.Evolog.Diff):
			return func() tea.Msg {
				selectedCommitId := o.getSelectedEvolog().CommitId
				output, _ := o.context.RunCommandImmediate(jj.Diff(selectedCommitId, ""))
				return common.ShowDiffMsg(output)
			}
		case key.Matches(msg, o.keyMap.Evolog.Restore):
			o.mode = restoreMode
			revisionsViewId := view.RevisionsViewId
			o.KeyDelegation = &revisionsViewId
		}
	case restoreMode:
		switch {
		case key.Matches(msg, o.keyMap.Cancel):
			o.mode = selectMode
			o.KeyDelegation = nil
			return nil
		case key.Matches(msg, o.keyMap.Apply):
			from := o.getSelectedEvolog().CommitId
			if current := o.context.Revisions.Current(); current != nil {
				target := current.Commit
				into := target.GetChangeId()
				o.ViewManager.UnregisterView(o.Id)
				return o.context.RunCommand(jj.RestoreEvolog(from, into), common.Refresh)
			}
		}
	}
	return nil
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

func (o *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateEvologMsg:
		o.SetItems(msg.rows)
		o.Cursor = 0
		return o, nil
	case tea.KeyMsg:
		return o, o.HandleKey(msg)
	}
	return o, nil
}

func (o *Operation) getSelectedEvolog() *models.Commit {
	return o.Items[o.Cursor].Commit
}

func (o *Operation) View() string {
	if len(o.Items) == 0 {
		return "loading"
	}
	h := min(o.Height-5, len(o.Items)*2)
	o.renderer.SetHeight(h)
	content := o.renderer.Render()
	content = lipgloss.PlaceHorizontal(o.Width, lipgloss.Left, content)
	return content
}

func (o *Operation) Render(commit *models.Commit, pos operations.RenderPosition) string {
	current := o.context.Revisions.Current()
	if current == nil {
		return ""
	}

	target := current.Commit
	if o.mode == restoreMode && pos == operations.RenderPositionBefore && target != nil && target.GetChangeId() == commit.GetChangeId() {
		selectedCommitId := o.getSelectedEvolog().CommitId
		return lipgloss.JoinHorizontal(0,
			o.markerStyle.Render("<< restore >>"),
			o.dimmedStyle.PaddingLeft(1).Render("restore from "),
			o.commitIdStyle.Render(selectedCommitId),
			o.dimmedStyle.Render(" into "),
			o.changeIdStyle.Render(target.GetChangeId()),
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
	return o.View()
}

func (o *Operation) load() tea.Msg {
	output, _ := o.context.RunCommandImmediate(jj.Evolog(o.revision.GetChangeId()))
	rows := parser.ParseRows(bytes.NewReader(output))
	return updateEvologMsg{
		rows: rows,
	}
}

func NewOperation(ctx *context.MainContext, revision *models.Commit, width int, height int) *Operation {
	//size := view.NewSizeable(width, height)
	l := ctx.Evolog
	el := &EvologList{
		List:          l,
		selectedStyle: common.DefaultPalette.Get("evolog selected"),
		textStyle:     common.DefaultPalette.Get("evolog text"),
		dimmedStyle:   common.DefaultPalette.Get("evolog dimmed"),
		commitIdStyle: common.DefaultPalette.Get("evolog commit_id"),
		changeIdStyle: common.DefaultPalette.Get("evolog change_id"),
		markerStyle:   common.DefaultPalette.Get("evolog target_marker"),
	}
	el.renderer = list.NewRenderer[*models.RevisionItem](l, el, view.NewSizeable(width, height))
	o := &Operation{
		//Sizeable:   size,
		EvologList: el,
		context:    ctx,
		keyMap:     config.Current.GetKeyMap(),
		revision:   revision,
	}
	return o
}
