package evolog

import (
	"bytes"

	tea "github.com/charmbracelet/bubbletea"
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
	rows []parser.Row
}

type mode int

const (
	selectMode mode = iota
	restoreMode
)

var _ list.IList = (*Operation)(nil)
var _ operations.Operation = (*Operation)(nil)
var _ view.IHasActionMap = (*Operation)(nil)

type Operation struct {
	*common.Sizeable
	context  *context.MainContext
	renderer *list.ListRenderer
	revision *jj.Commit
	mode     mode
	rows     []parser.Row
	cursor   int
	keyMap   config.KeyMappings[key.Binding]
	target   *jj.Commit
	styles   styles
}

func (o *Operation) GetActionMap() map[string]common.Action {
	if o.mode == selectMode {
		return map[string]common.Action{
			"j":   {Id: "evolog.down"},
			"k":   {Id: "evolog.up"},
			"esc": {Id: "close evolog"},
			"d":   {Id: "evolog.diff"},
			"r":   {Id: "evolog.restore"},
		}
	} else {
		return map[string]common.Action{
			"esc": {Id: "close evolog"},
			"enter": {Id: "evolog.apply", Next: []common.Action{
				{Id: "close evolog"},
				{Id: "ui.refresh"},
			}},
		}
	}
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
	if msg, ok := msg.(common.InvokeActionMsg); ok {
		switch msg.Action.Id {
		case "evolog.up":
			if o.cursor > 0 {
				o.cursor--
			}
			return o, nil
		case "evolog.down":
			if o.cursor < len(o.rows)-1 {
				o.cursor++
			}
			return o, nil
		case "evolog.restore":
			o.mode = restoreMode
		case "evolog.diff":
			return o, tea.Sequence(common.InvokeAction(common.Action{Id: "ui.diff"}), func() tea.Msg {
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
		o.cursor = 0
		return o, o.updateSelection()
	}
	return o, nil
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
		keyMap:   config.Current.GetKeyMap(),
		revision: revision,
		rows:     nil,
		cursor:   0,
		styles:   styles,
	}
	o.renderer = list.NewRenderer(o, common.NewSizeable(width, height))
	return o
}
