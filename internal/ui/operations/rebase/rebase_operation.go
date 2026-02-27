package rebase

import (
	"fmt"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/target_picker"
	"github.com/idursun/jjui/internal/ui/render"
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
	_ common.Overlay       = (*Operation)(nil)
	_ common.Editable      = (*Operation)(nil)
)

type Operation struct {
	context        *context.MainContext
	From           jj.SelectedRevisions
	InsertStart    *jj.Commit
	To             *jj.Commit
	Source         Source
	Target         Target
	targetName     string
	targetPicker   *target_picker.Model
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

func (r *Operation) IsEditing() bool {
	return r.targetPicker != nil
}

func (r *Operation) IsOverlay() bool {
	return r.targetPicker != nil
}

func (r *Operation) Init() tea.Cmd {
	return nil
}

func (r *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case target_picker.TargetSelectedMsg:
		r.targetPicker = nil
		r.targetName = strings.TrimSpace(msg.Target)
		return r.handleIntent(intents.Apply{Force: msg.Force})
	case target_picker.TargetPickerCancelMsg:
		r.targetPicker = nil
		return nil
	case updateHighlightedIdsMsg:
		r.highlightedIds = msg.ids
		return nil
	case intents.Intent:
		if r.targetPicker != nil {
			switch msg.(type) {
			case intents.TargetPickerNavigate, intents.TargetPickerApply, intents.TargetPickerCancel:
				return r.targetPicker.Update(msg)
			}
		}
		return r.handleIntent(msg)
	case tea.KeyMsg:
		if r.targetPicker != nil {
			return r.targetPicker.Update(msg)
		}
		return nil
	default:
		if r.targetPicker != nil {
			return r.targetPicker.Update(msg)
		}
		return nil
	}
}

func (r *Operation) handleIntent(intent intents.Intent) tea.Cmd {
	switch msg := intent.(type) {
	case intents.StartAceJump:
		return common.StartAceJump()
	case intents.RebaseSetSource:
		r.Source = rebaseSourceFromIntent(msg.Source)
	case intents.RebaseSetTarget:
		r.Target = rebaseTargetFromIntent(msg.Target)
		if r.Target == TargetInsert {
			r.InsertStart = r.To
		}
	case intents.RebaseOpenTargetPicker:
		r.targetPicker = target_picker.NewModel(r.context)
		return r.targetPicker.Init()
	case intents.RebaseToggleSkipEmptied:
		r.SkipEmptied = !r.SkipEmptied
	case intents.Apply:
		skipEmptied := r.SkipEmptied
		if r.Target == TargetInsert {
			insertAfter := r.InsertStart.GetChangeId()
			insertBefore := r.targetArg()
			return r.context.RunCommand(jj.RebaseInsert(r.From, insertAfter, insertBefore, skipEmptied, msg.Force), common.RefreshAndSelect(r.From.Last()), common.Close)
		}
		source := sourceToFlags[r.Source]
		target := targetToFlags[r.Target]
		return r.context.RunCommand(jj.Rebase(r.From, r.targetArg(), source, target, skipEmptied, msg.Force), common.RefreshAndSelect(r.From.Last()), common.Close)
	case intents.Cancel:
		return common.Close
	default:
		return nil
	}
	return nil
}

func (r *Operation) ResolveAction(action keybindings.Action, args map[string]any) (intents.Intent, bool) {
	return actions.ResolveByScopeStrict(r.Scope(), action, args)
}

func rebaseSourceFromIntent(source intents.RebaseSource) Source {
	switch source {
	case intents.RebaseSourceRevision:
		return SourceRevision
	case intents.RebaseSourceBranch:
		return SourceBranch
	case intents.RebaseSourceDescendants:
		return SourceDescendants
	default:
		return SourceRevision
	}
}

func rebaseTargetFromIntent(target intents.RebaseTarget) Target {
	switch target {
	case intents.RebaseTargetDestination:
		return TargetDestination
	case intents.RebaseTargetAfter:
		return TargetAfter
	case intents.RebaseTargetBefore:
		return TargetBefore
	case intents.RebaseTargetInsert:
		return TargetInsert
	default:
		return TargetDestination
	}
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
		return updateHighlightedIdsMsg{ids: ids}
	})
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

func (r *Operation) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}

func (r *Operation) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
}

func (r *Operation) Name() string {
	return "rebase"
}

func (r *Operation) Scope() keybindings.Scope {
	if r.targetPicker != nil {
		return keybindings.Scope(actions.OwnerTargetPicker)
	}
	return keybindings.Scope(actions.OwnerRebase)
}

func (r *Operation) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if r.targetPicker != nil {
		r.targetPicker.ViewRect(dl, box)
	}
}

func (r *Operation) targetArg() string {
	if strings.TrimSpace(r.targetName) != "" {
		return r.targetName
	}
	if r.To != nil {
		return r.To.GetChangeId()
	}
	return ""
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
		From:    from,
		Source:  source,
		Target:  target,
		styles:  styles,
	}
}
