package revert

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

type Target int

const (
	TargetDestination Target = iota
	TargetAfter
	TargetBefore
	TargetInsert
)

var (
	targetToFlags = map[Target]string{
		TargetAfter:       "--insert-after",
		TargetBefore:      "--insert-before",
		TargetDestination: "--destination",
	}
)

type styles struct {
	shortcut     lipgloss.Style
	dimmed       lipgloss.Style
	sourceMarker lipgloss.Style
	targetMarker lipgloss.Style
	changeId     lipgloss.Style
	text         lipgloss.Style
}

var _ operations.Operation = (*Operation)(nil)
var _ view.IHasActionMap = (*Operation)(nil)
var _ view.ICommandBuilder = (*Operation)(nil)

type Operation struct {
	context        *context.MainContext
	From           jj.SelectedRevisions
	InsertStart    *jj.Commit
	To             *jj.Commit
	Target         Target
	highlightedIds []string
	styles         styles
}

func (r *Operation) GetCommand() jj.CommandArgs {
	if r.Target == TargetInsert {
		return jj.RevertInsert(r.From, r.InsertStart.GetChangeId(), r.To.GetChangeId())
	} else {
		source := "--revisions"
		target := targetToFlags[r.Target]
		return jj.Revert(r.From, r.To.GetChangeId(), source, target)
	}
}

func (r *Operation) GetActionMap() actions.ActionMap {
	return config.Current.GetBindings("revert")
}

func (r *Operation) Init() tea.Cmd {
	return nil
}

func (r *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		switch msg.Action.Id {
		case "revert.onto":
			r.Target = TargetDestination
		case "revert.after":
			r.Target = TargetAfter
		case "revert.before":
			r.Target = TargetBefore
		case "revert.insert":
			r.Target = TargetInsert
			r.InsertStart = r.To
		case "revert.apply":
			if r.Target == TargetInsert {
				return r, r.context.RunCommand(jj.RevertInsert(r.From, r.InsertStart.GetChangeId(), r.To.GetChangeId()), common.RefreshAndSelect(r.From.Last()))
			} else {
				source := "--revisions"
				target := targetToFlags[r.Target]
				return r, r.context.RunCommand(jj.Revert(r.From, r.To.GetChangeId(), source, target), common.RefreshAndSelect(r.From.Last()))
			}
		}
	}
	return r, nil
}

func (r *Operation) View() string {
	return ""
}

func (r *Operation) SetSelectedRevision(commit *jj.Commit) {
	r.highlightedIds = nil
	r.To = commit
	r.highlightedIds = r.From.GetIds()
}

func (r *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos == operations.RenderBeforeChangeId {
		changeId := commit.GetChangeId()
		if slices.Contains(r.highlightedIds, changeId) {
			return r.styles.sourceMarker.Render("<< revert >>")
		}
		if r.Target == TargetInsert && r.InsertStart.GetChangeId() == commit.GetChangeId() {
			return r.styles.sourceMarker.Render("<< after this >>")
		}
		if r.Target == TargetInsert && r.To.GetChangeId() == commit.GetChangeId() {
			return r.styles.sourceMarker.Render("<< before this >>")
		}
		return ""
	}
	expectedPos := operations.RenderPositionBefore
	if r.Target == TargetBefore || r.Target == TargetInsert {
		expectedPos = operations.RenderPositionAfter
	}

	if pos != expectedPos {
		return ""
	}

	isSelected := r.To != nil && r.To.GetChangeId() == commit.GetChangeId()
	if !isSelected {
		return ""
	}

	var source string
	isMany := len(r.From.Revisions) > 0
	switch {
	case isMany:
		source = "revisions "
	default:
		source = "revision "
	}
	var ret string
	if r.Target == TargetDestination {
		ret = "onto"
	}
	if r.Target == TargetAfter {
		ret = "after"
	}
	if r.Target == TargetBefore {
		ret = "before"
	}
	if r.Target == TargetInsert {
		ret = "insert"
	}

	if r.Target == TargetInsert {
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			r.styles.targetMarker.Render("<< insert >>"),
			" ",
			r.styles.dimmed.Render(source),
			r.styles.changeId.Render(strings.Join(r.From.GetIds(), " ")),
			r.styles.dimmed.Render(" between "),
			r.styles.changeId.Render(r.InsertStart.GetChangeId()),
			r.styles.dimmed.Render(" and "),
			r.styles.changeId.Render(r.To.GetChangeId()),
		)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		r.styles.targetMarker.Render("<< "+ret+" >>"),
		r.styles.dimmed.Render(" revert "),
		r.styles.dimmed.Render(source),
		r.styles.changeId.Render(strings.Join(r.From.GetIds(), " ")),
		r.styles.dimmed.Render(" "),
		r.styles.dimmed.Render(ret),
		r.styles.dimmed.Render(" "),
		r.styles.changeId.Render(r.To.GetChangeId()),
	)
}

func (r *Operation) Name() string {
	return "revert"
}

func NewOperation(context *context.MainContext, from jj.SelectedRevisions, target Target) *Operation {
	styles := styles{
		changeId:     common.DefaultPalette.Get("revert change_id"),
		shortcut:     common.DefaultPalette.Get("revert shortcut"),
		dimmed:       common.DefaultPalette.Get("revert dimmed"),
		sourceMarker: common.DefaultPalette.Get("revert source_marker"),
		targetMarker: common.DefaultPalette.Get("revert target_marker"),
	}
	return &Operation{
		context: context,
		From:    from,
		Target:  target,
		styles:  styles,
	}
}
