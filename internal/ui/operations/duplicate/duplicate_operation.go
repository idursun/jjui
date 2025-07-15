package duplicate

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type Target int

const (
	TargetDestination Target = iota
	TargetAfter
	TargetBefore
)

var (
	targetToFlags = map[Target]string{
		TargetAfter:       "--insert-after",
		TargetBefore:      "--insert-before",
		TargetDestination: "--destination",
	}
)

type Operation struct {
	context        *appContext.MainContext
	From           jj.SelectedRevisions
	InsertStart    *jj.Commit
	To             *jj.Commit
	Target         Target
	keyMap         config.KeyMappings[key.Binding]
	highlightedIds []string
	styles         styles
}

type styles struct {
	changeId     lipgloss.Style
	dimmed       lipgloss.Style
	shortcut     lipgloss.Style
	targetMarker lipgloss.Style
	sourceMarker lipgloss.Style
}

func (r *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, r.keyMap.Duplicate.Onto):
		r.Target = TargetDestination
	case key.Matches(msg, r.keyMap.Duplicate.After):
		r.Target = TargetAfter
	case key.Matches(msg, r.keyMap.Duplicate.Before):
		r.Target = TargetBefore
	case key.Matches(msg, r.keyMap.Apply):
		target := targetToFlags[r.Target]
		return r.context.RunCommand(jj.Duplicate(r.From, r.To.GetChangeId(), target), common.RefreshAndSelect(r.From.Last()), common.Close)
	case key.Matches(msg, r.keyMap.Cancel):
		return common.Close
	}
	return nil
}

func (r *Operation) SetSelectedRevision(commit *jj.Commit) {
	r.highlightedIds = nil
	r.To = commit
	revset := ""
	if output, err := r.context.RunCommandImmediate(jj.GetIdsFromRevset(revset)); err == nil {
		ids := strings.Split(strings.TrimSpace(string(output)), "\n")
		r.highlightedIds = ids
	}
}

func (r *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		r.keyMap.Cancel,
		r.keyMap.Duplicate.After,
		r.keyMap.Duplicate.Before,
		r.keyMap.Duplicate.Onto,
	}
}

func (r *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{r.ShortHelp()}
}

func (r *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos == operations.RenderBeforeChangeId {
		changeId := commit.GetChangeId()
		if slices.Contains(r.highlightedIds, changeId) {
			return r.styles.sourceMarker.Render("<< move >>") + " "
		}
		return ""
	}
	expectedPos := operations.RenderPositionBefore
	if r.Target == TargetBefore {
		expectedPos = operations.RenderPositionAfter
	}

	if pos != expectedPos {
		return ""
	}

	isSelected := r.To != nil && r.To.GetChangeId() == commit.GetChangeId()
	if !isSelected {
		return ""
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

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		r.styles.targetMarker.Render("<< "+ret+" >>"),
		" ",
		r.styles.dimmed.Render("duplicate"),
		" ",
		r.styles.changeId.Render(strings.Join(r.From.GetIds(), " ")),
		" ",
		r.styles.dimmed.Render(ret),
		" ",
		r.styles.changeId.Render(r.To.GetChangeId()),
	)
}

func (r *Operation) Name() string {
	return "duplicate"
}

func NewOperation(context *appContext.MainContext, from jj.SelectedRevisions, target Target) *Operation {
	styles := styles{
		changeId:     common.DefaultPalette.Get("change_id"),
		dimmed:       common.DefaultPalette.Get("duplicate dimmed"),
		sourceMarker: common.DefaultPalette.Get("duplicate source_marker"),
		targetMarker: common.DefaultPalette.Get("duplicate target_marker"),
	}
	return &Operation{
		context: context,
		keyMap:  config.Current.GetKeyMap(),
		From:    from,
		Target:  target,
		styles:  styles,
	}
}
