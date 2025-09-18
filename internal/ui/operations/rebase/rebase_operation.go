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
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/view"
)

var _ tea.Model = (*Operation)(nil)
var _ view.IViewModel = (*Operation)(nil)
var _ view.IView = (*Operation)(nil)
var _ operations.Operation = (*Operation)(nil)

type Operation struct {
	*view.ViewNode
	context        *context.MainContext
	From           jj.SelectedRevisions
	InsertStart    *models.RevisionItem
	To             *models.RevisionItem
	Source         jj.Source
	Target         jj.Target
	keyMap         config.KeyMappings[key.Binding]
	highlightedIds []string
	styles         styles
}

func (r *Operation) Mount(v *view.ViewNode) {
	r.ViewNode = v
	v.Id = r.GetId()
	keyDelegatedViewId := view.RevisionsViewId
	v.KeyDelegation = &keyDelegatedViewId
	v.NeedsRefresh = true
}

func (r *Operation) GetId() view.ViewId {
	return "rebase"
}

func (r *Operation) Init() tea.Cmd {
	return nil
}

func (r *Operation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if cmd := r.HandleKey(msg); cmd != nil {
			return r, cmd
		}
	case common.RefreshMsg:
		r.setSelectedRevision()
		return r, nil
	}
	return r, nil
}

func (r *Operation) View() string {
	return ""
}

type styles struct {
	shortcut     lipgloss.Style
	dimmed       lipgloss.Style
	sourceMarker lipgloss.Style
	targetMarker lipgloss.Style
	changeId     lipgloss.Style
	text         lipgloss.Style
}

func (r *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, r.keyMap.Rebase.Revision):
		r.Source = jj.SourceRevision
	case key.Matches(msg, r.keyMap.Rebase.Branch):
		r.Source = jj.SourceBranch
	case key.Matches(msg, r.keyMap.Rebase.Source):
		r.Source = jj.SourceDescendants
	case key.Matches(msg, r.keyMap.Rebase.Onto):
		r.Target = jj.TargetDestination
	case key.Matches(msg, r.keyMap.Rebase.After):
		r.Target = jj.TargetAfter
	case key.Matches(msg, r.keyMap.Rebase.Before):
		r.Target = jj.TargetBefore
	case key.Matches(msg, r.keyMap.Rebase.Insert):
		r.Target = jj.TargetInsert
		r.InsertStart = r.To
	case key.Matches(msg, r.keyMap.Apply, r.keyMap.ForceApply):
		r.ViewManager.UnregisterView(r.GetId())
		ignoreImmutable := key.Matches(msg, r.keyMap.ForceApply)
		if r.Target == jj.TargetInsert {
			return r.context.RunCommand(jj.Args(jj.RebaseInsertArgs{From: r.From, InsertAfter: *r.InsertStart, InsertBefore: *r.To, IgnoreImmutable: ignoreImmutable}), common.RefreshAndSelect(r.From.Last()))
		} else {
			return r.context.RunCommand(jj.Args(jj.RebaseCommandArgs{From: r.From, To: *r.To, Source: r.Source, Target: r.Target, IgnoreImmutable: ignoreImmutable}), common.RefreshAndSelect(r.From.Last()))
		}
	case key.Matches(msg, r.keyMap.Cancel):
		r.ViewManager.UnregisterView(r.GetId())
		return nil
	}
	return nil
}

func (r *Operation) setSelectedRevision() {
	current := r.context.Revisions.Current()
	if current == nil {
		return
	}
	r.highlightedIds = nil
	r.To = current
	revset := ""
	switch r.Source {
	case jj.SourceRevision:
		r.highlightedIds = r.From.GetIds()
		return
	case jj.SourceBranch:
		revset = fmt.Sprintf("(%s..(%s))::", r.To.Commit.GetChangeId(), strings.Join(r.From.GetIds(), "|"))
	case jj.SourceDescendants:
		revset = fmt.Sprintf("(%s)::", strings.Join(r.From.GetIds(), "|"))
	}
	if output, err := r.context.RunCommandImmediate(jj.GetIdsFromRevset(revset).GetArgs()); err == nil {
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

func (r *Operation) Render(commit *models.Commit, pos operations.RenderPosition) string {
	if pos == operations.RenderBeforeChangeId {
		changeId := commit.GetChangeId()
		if slices.Contains(r.highlightedIds, changeId) {
			return r.styles.sourceMarker.Render("<< move >>")
		}
		if r.Target == jj.TargetInsert && r.InsertStart.Commit.GetChangeId() == commit.GetChangeId() {
			return r.styles.sourceMarker.Render("<< after this >>")
		}
		if r.Target == jj.TargetInsert && r.To.Commit.GetChangeId() == commit.GetChangeId() {
			return r.styles.sourceMarker.Render("<< before this >>")
		}
		return ""
	}
	expectedPos := operations.RenderPositionBefore
	if r.Target == jj.TargetBefore || r.Target == jj.TargetInsert {
		expectedPos = operations.RenderPositionAfter
	}

	if pos != expectedPos {
		return ""
	}

	isSelected := r.To != nil && r.To.Commit.GetChangeId() == commit.GetChangeId()
	if !isSelected {
		return ""
	}

	var source string
	isMany := len(r.From) > 0
	switch {
	case r.Source == jj.SourceBranch && isMany:
		source = "branches of "
	case r.Source == jj.SourceBranch:
		source = "branch of "
	case r.Source == jj.SourceDescendants && isMany:
		source = "itself and descendants of each "
	case r.Source == jj.SourceDescendants:
		source = "itself and descendants of "
	case r.Source == jj.SourceRevision && isMany:
		source = "revisions "
	case r.Source == jj.SourceRevision:
		source = "revision "
	}
	var ret string
	if r.Target == jj.TargetDestination {
		ret = "onto"
	}
	if r.Target == jj.TargetAfter {
		ret = "after"
	}
	if r.Target == jj.TargetBefore {
		ret = "before"
	}
	if r.Target == jj.TargetInsert {
		ret = "insert"
	}

	if r.Target == jj.TargetInsert {
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			r.styles.targetMarker.Render("<< insert >>"),
			" ",
			r.styles.dimmed.Render(source),
			r.styles.changeId.Render(strings.Join(r.From.GetIds(), " ")),
			r.styles.dimmed.Render(" between "),
			r.styles.changeId.Render(r.InsertStart.Commit.GetChangeId()),
			r.styles.dimmed.Render(" and "),
			r.styles.changeId.Render(r.To.Commit.GetChangeId()),
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
		r.styles.changeId.Render(r.To.Commit.GetChangeId()),
	)
}

func NewOperation(context *context.MainContext, from jj.SelectedRevisions) view.IViewModel {
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
		Source:  jj.SourceRevision,
		Target:  jj.TargetDestination,
		styles:  styles,
	}
}
