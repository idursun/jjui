package evolog

import (
	"bytes"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/list"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/context"
)

type updateEvologMsg struct {
	rows []parser.Row
}

type mode int

const (
	selectMode mode = iota
	restoreMode
)

var _ list.IList = (*Operation)(nil)
var _ list.IListCursor = (*Operation)(nil)
var _ operations.Operation = (*Operation)(nil)
var _ view.IHasActionMap = (*Operation)(nil)
var _ common.ContextProvider = (*Operation)(nil)

type Operation struct {
	*common.Sizeable
	context  *context.MainContext
	renderer *list.ListRenderer
	revision *jj.Commit
	mode     mode
	rows     []parser.Row
	cursor   int
	target   *jj.Commit
	styles   styles
}

func (o *Operation) Read(value string) string {
	switch value {
	case jj.CommitIdPlaceholder:
		if selectedEvolog := o.getSelectedEvolog(); selectedEvolog != nil {
			return selectedEvolog.CommitId
		}
	}
	return ""
}

func (o *Operation) Cursor() int {
	return o.cursor
}

func (o *Operation) SetCursor(index int) {
	if index < 0 || index >= len(o.rows) {
		return
	}
	o.cursor = index
	o.context.Router.ContinueAction("@evolog.cursor")
}

func (o *Operation) GetActionMap() actions.ActionMap {
	if o.mode == restoreMode {
		return config.Current.GetBindings("evolog.restore")
	}

	return config.Current.GetBindings("evolog")
}

func (o *Operation) Init() tea.Cmd {
	return o.load
}

func (o *Operation) View() string {
	if len(o.rows) == 0 {
		return "loading"
	}
	o.renderer.SetWidth(o.Width)
	o.renderer.SetHeight(min(o.Height-5, len(o.rows)*2))
	content := o.renderer.Render(o.cursor)
	content = lipgloss.PlaceHorizontal(o.Width, lipgloss.Left, content)
	return content
}

func (o *Operation) Len() int {
	return len(o.rows)
}

func (o *Operation) GetItemRenderer(index int) list.IItemRenderer {
	row := o.rows[index]
	selected := index == o.cursor
	styleOverride := o.styles.textStyle
	if selected {
		styleOverride = o.styles.selectedStyle
	}
	return &itemRenderer{
		row:           row,
		styleOverride: styleOverride,
	}
}

type styles struct {
	dimmedStyle   lipgloss.Style
	commitIdStyle lipgloss.Style
	changeIdStyle lipgloss.Style
	markerStyle   lipgloss.Style
	textStyle     lipgloss.Style
	selectedStyle lipgloss.Style
}

func (o *Operation) SetSelectedRevision(commit *jj.Commit) {
	o.target = commit
}

func (o *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(actions.InvokeActionMsg); ok {
		switch msg.Action.Id {
		case "evolog.up":
			o.SetCursor(o.cursor - 1)
			return o, nil
		case "evolog.down":
			o.SetCursor(o.cursor + 1)
			return o, nil
		case "evolog.restore":
			o.mode = restoreMode
		case "evolog.diff":
			return o, tea.Sequence(actions.InvokeAction(actions.Action{Id: "ui.diff"}), func() tea.Msg {
				output, _ := o.context.RunCommandImmediate(jj.Diff(o.getSelectedEvolog().CommitId, ""))
				return common.ShowDiffMsg(output)
			})
		case "evolog.apply":
			from := o.getSelectedEvolog().CommitId
			into := o.target.GetChangeId()
			return o, o.context.RunCommand(jj.RestoreEvolog(from, into))
		}
	}
	switch msg := msg.(type) {
	case updateEvologMsg:
		o.rows = msg.rows
		o.SetCursor(0)
		return o, nil
	}
	return o, nil
}

func (o *Operation) getSelectedEvolog() *jj.Commit {
	if len(o.rows) == 0 || o.cursor < 0 || o.cursor >= len(o.rows) {
		return nil
	}
	return o.rows[o.Cursor()].Commit
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
	return o.View()
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

func NewOperation(context *context.MainContext, revision *jj.Commit, width int, height int) *Operation {
	styles := styles{
		dimmedStyle:   common.DefaultPalette.Get("evolog dimmed"),
		commitIdStyle: common.DefaultPalette.Get("evolog commit_id"),
		changeIdStyle: common.DefaultPalette.Get("evolog change_id"),
		markerStyle:   common.DefaultPalette.Get("evolog target_marker"),
		textStyle:     common.DefaultPalette.Get("evolog text"),
		selectedStyle: common.DefaultPalette.Get("evolog selected"),
	}
	o := &Operation{
		Sizeable: &common.Sizeable{Width: width, Height: height},
		context:  context,
		revision: revision,
		rows:     nil,
		cursor:   0,
		styles:   styles,
	}
	o.renderer = list.NewRenderer(o, common.NewSizeable(width, height))
	return o
}
