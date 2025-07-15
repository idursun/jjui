package rebase

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
)

type Source int

const (
	SourceRevision Source = iota
	SourceBranch
	SourceDescendants
)

type Target int

const (
	TargetDestination Target = iota
	TargetAfter
	TargetBefore
	TargetInsert
)

var (
	sourceToFlags = map[Source]string{
		SourceBranch:      "--branch",
		SourceRevision:    "--revisions",
		SourceDescendants: "--source",
	}
	targetToFlags = map[Target]string{
		TargetAfter:       "--insert-after",
		TargetBefore:      "--insert-before",
		TargetDestination: "--destination",
	}
)

type Operation struct {
	context        *context.MainContext
	From           jj.SelectedRevisions
	InsertStart    *jj.Commit
	To             *jj.Commit
	Source         Source
	Target         Target
	keyMap         config.KeyMappings[key.Binding]
	highlightedIds []string
}

func (r *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, r.keyMap.Rebase.Revision):
		r.Source = SourceRevision
	case key.Matches(msg, r.keyMap.Rebase.Branch):
		r.Source = SourceBranch
	case key.Matches(msg, r.keyMap.Rebase.Source):
		r.Source = SourceDescendants
	case key.Matches(msg, r.keyMap.Rebase.Onto):
		r.Target = TargetDestination
	case key.Matches(msg, r.keyMap.Rebase.After):
		r.Target = TargetAfter
	case key.Matches(msg, r.keyMap.Rebase.Before):
		r.Target = TargetBefore
	case key.Matches(msg, r.keyMap.Rebase.Insert):
		r.Target = TargetInsert
		r.InsertStart = r.To
	case key.Matches(msg, r.keyMap.Apply):
		if r.Target == TargetInsert {
			return r.context.RunCommand(jj.RebaseInsert(r.From, r.InsertStart.GetChangeId(), r.To.GetChangeId()), common.RefreshAndSelect(r.From.Last()), common.Close)
		} else {
			source := sourceToFlags[r.Source]
			target := targetToFlags[r.Target]
			return r.context.RunCommand(jj.Rebase(r.From, r.To.GetChangeId(), source, target), common.RefreshAndSelect(r.From.Last()), common.Close)
		}
	case key.Matches(msg, r.keyMap.Cancel):
		return common.Close
	}
	return nil
}

func (r *Operation) SetSelectedRevision(commit *jj.Commit) {
	r.highlightedIds = nil
	r.To = commit
	revset := ""
	switch r.Source {
	case SourceRevision:
		r.highlightedIds = r.From.GetIds()
		return
	case SourceBranch:
		revset = fmt.Sprintf("(%s..(%s))::", r.To.GetChangeId(), strings.Join(r.From.GetIds(), "|"))
	case SourceDescendants:
		revset = fmt.Sprintf("(%s)::", strings.Join(r.From.GetIds(), "|"))
	}
	if output, err := r.context.RunCommandImmediate(jj.GetIdsFromRevset(revset)); err == nil {
		ids := strings.Split(strings.TrimSpace(string(output)), "\n")
		r.highlightedIds = ids
	}
}

func (r *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		r.keyMap.Rebase.Revision,
		r.keyMap.Rebase.Branch,
		r.keyMap.Rebase.Source,
		r.keyMap.Rebase.Before,
		r.keyMap.Rebase.After,
		r.keyMap.Rebase.Onto,
		r.keyMap.Rebase.Insert,
	}
}

func (r *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{r.ShortHelp()}
}

func (r *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos == operations.RenderBeforeChangeId {
		changeId := commit.GetChangeId()
		if slices.Contains(r.highlightedIds, changeId) {
			return common.DefaultPalette.SourceMarker.Render("<< move >>") + " "
		}
		if r.Target == TargetInsert && r.InsertStart.GetChangeId() == commit.GetChangeId() {
			return common.DefaultPalette.SourceMarker.Render("<< after this >>") + " "
		}
		if r.Target == TargetInsert && r.To.GetChangeId() == commit.GetChangeId() {
			return common.DefaultPalette.SourceMarker.Render("<< before this >>") + " "
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
	case r.Source == SourceBranch && isMany:
		source = "branches of "
	case r.Source == SourceBranch:
		source = "branch of "
	case r.Source == SourceDescendants && isMany:
		source = "itself and descendants of each "
	case r.Source == SourceDescendants:
		source = "itself and descendants of "
	case r.Source == SourceRevision && isMany:
		source = "revisions "
	case r.Source == SourceRevision:
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
			common.DefaultPalette.TargetMarker.Render("<< insert >>"),
			" ",
			common.DefaultPalette.Dimmed.Render(source),
			common.DefaultPalette.ChangeId.Render(strings.Join(r.From.GetIds(), " ")),
			common.DefaultPalette.Dimmed.Render(" between "),
			common.DefaultPalette.ChangeId.Render(r.InsertStart.GetChangeId()),
			common.DefaultPalette.Dimmed.Render(" and "),
			common.DefaultPalette.ChangeId.Render(r.To.GetChangeId()),
		)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		common.DefaultPalette.TargetMarker.Render("<< "+ret+" >>"),
		" ",
		common.DefaultPalette.Dimmed.Render("rebase"),
		" ",
		common.DefaultPalette.Dimmed.Render(source),
		common.DefaultPalette.ChangeId.Render(strings.Join(r.From.GetIds(), " ")),
		" ",
		common.DefaultPalette.Dimmed.Render(ret),
		" ",
		common.DefaultPalette.ChangeId.Render(r.To.GetChangeId()),
	)
}

func (r *Operation) Name() string {
	return "rebase"
}

func NewOperation(context *context.MainContext, from jj.SelectedRevisions, source Source, target Target) *Operation {
	return &Operation{
		context: context,
		keyMap:  config.Current.GetKeyMap(),
		From:    from,
		Source:  source,
		Target:  target,
	}
}
