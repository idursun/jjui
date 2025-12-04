package rebase

import (
	"fmt"
	"slices"
	"strings"
	"time"

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
		TargetDestination: "--onto",
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

var (
	_ operations.Operation = (*Operation)(nil)
	_ common.Focusable     = (*Operation)(nil)
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
	styles         styles
	SkipEmptied    bool
}

type updateHighlightedIdsMsg struct {
	ids []string
}

const debounceDuration = 250 * time.Millisecond

func (r *Operation) IsFocused() bool {
	return true
}

func (r *Operation) Init() tea.Cmd {
	return nil
}

func (r *Operation) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(updateHighlightedIdsMsg); ok {
		r.highlightedIds = msg.ids
		return nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		return r.HandleKey(msg)
	}
	return nil
}

func (r *Operation) View() string {
	return ""
}

func (r *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, r.keyMap.AceJump):
		return common.StartAceJump()
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
	case key.Matches(msg, r.keyMap.Rebase.Insert):
		r.Target = TargetInsert
		r.InsertStart = r.To
	case key.Matches(msg, r.keyMap.Rebase.SkipEmptied):
		r.SkipEmptied = !r.SkipEmptied
	case key.Matches(msg, r.keyMap.Apply, r.keyMap.ForceApply):
		ignoreImmutable := key.Matches(msg, r.keyMap.ForceApply)
		skipEmptied := r.SkipEmptied
		if r.Target == TargetInsert {
			return r.context.RunCommand(jj.RebaseInsert(r.From, r.InsertStart.GetChangeId(), r.To.GetChangeId(), skipEmptied, ignoreImmutable), common.RefreshAndSelect(r.From.Last()), common.Close)
		} else {
			source := sourceToFlags[r.Source]
			target := targetToFlags[r.Target]
			return r.context.RunCommand(jj.Rebase(r.From, r.To.GetChangeId(), source, target, skipEmptied, ignoreImmutable), common.RefreshAndSelect(r.From.Last()), common.Close)
		}
	case key.Matches(msg, r.keyMap.Cancel):
		return common.Close
	}
	return nil
}

func (r *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	r.To = commit
	identifier := fmt.Sprintf("rebase-highlight-%p", r)

	revset := ""
	switch r.Source {
	case SourceRevision:
		r.highlightedIds = r.From.GetIds()
		return nil
	case SourceBranch:
		revset = fmt.Sprintf("(%s..(%s))::", r.To.GetChangeId(), strings.Join(r.From.GetIds(), "|"))
	case SourceDescendants:
		revset = fmt.Sprintf("(%s)::", strings.Join(r.From.GetIds(), "|"))
	}

	return common.Debounce(identifier, debounceDuration, func() tea.Msg {
		output, err := r.context.RunCommandImmediate(jj.GetIdsFromRevset(revset))
		if err != nil {
			return nil
		}
		ids := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(ids) == 1 && ids[0] == "" {
			ids = nil
		}
		return common.DeferredUpdateMsg{Fn: func() tea.Cmd {
			return r.Update(updateHighlightedIdsMsg{ids: ids})
		}}
	})
}

func (r *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		r.keyMap.Apply,
		r.keyMap.ForceApply,
		r.keyMap.Rebase.Revision,
		r.keyMap.Rebase.Branch,
		r.keyMap.Rebase.Source,
		r.keyMap.Rebase.Before,
		r.keyMap.Rebase.After,
		r.keyMap.Rebase.Onto,
		r.keyMap.Rebase.Insert,
		r.keyMap.Rebase.SkipEmptied,
		r.keyMap.AceJump,
	}
}

func (r *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{r.ShortHelp()}
}

func (r *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos == operations.RenderBeforeChangeId {
		changeId := commit.GetChangeId()
		marker := ""
		if slices.Contains(r.highlightedIds, changeId) {
			marker = "<< move >>"
		}
		if r.Target == TargetInsert && r.InsertStart.GetChangeId() == commit.GetChangeId() {
			marker = "<< after this >>"
		}
		if r.Target == TargetInsert && r.To.GetChangeId() == commit.GetChangeId() {
			marker = "<< before this >>"
		}
		if r.SkipEmptied && marker != "" {
			marker += " (skip emptied)"
		}
		return r.styles.sourceMarker.Render(marker)
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
		r.styles.dimmed.Render(" rebase "),
		r.styles.dimmed.Render(source),
		r.styles.changeId.Render(strings.Join(r.From.GetIds(), " ")),
		r.styles.dimmed.Render(" "),
		r.styles.dimmed.Render(ret),
		r.styles.dimmed.Render(" "),
		r.styles.changeId.Render(r.To.GetChangeId()),
	)
}

func (r *Operation) Name() string {
	return "rebase"
}

func NewOperation(context *context.MainContext, from jj.SelectedRevisions, source Source, target Target) *Operation {
	styles := styles{
		changeId:     common.DefaultPalette.Get("rebase change_id"),
		shortcut:     common.DefaultPalette.Get("rebase shortcut"),
		dimmed:       common.DefaultPalette.Get("rebase dimmed"),
		sourceMarker: common.DefaultPalette.Get("rebase source_marker"),
		targetMarker: common.DefaultPalette.Get("rebase target_marker"),
	}
	return &Operation{
		context: context,
		keyMap:  config.Current.GetKeyMap(),
		From:    from,
		Source:  source,
		Target:  target,
		styles:  styles,
	}
}
