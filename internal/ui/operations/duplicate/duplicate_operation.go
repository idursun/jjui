package duplicate

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/target_picker"
	"github.com/idursun/jjui/internal/ui/render"
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
		TargetDestination: "--onto",
	}
)

type styles struct {
	changeId     lipgloss.Style
	dimmed       lipgloss.Style
	shortcut     lipgloss.Style
	targetMarker lipgloss.Style
	sourceMarker lipgloss.Style
}

var _ operations.Operation = (*Operation)(nil)
var _ common.Focusable = (*Operation)(nil)
var _ common.Overlay = (*Operation)(nil)
var _ common.Editable = (*Operation)(nil)

type Operation struct {
	context      *appContext.MainContext
	From         jj.SelectedRevisions
	InsertStart  *jj.Commit
	To           *jj.Commit
	Target       Target
	targetName   string
	targetPicker *target_picker.Model
	styles       styles
}

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
	case intents.DuplicateSetTarget:
		r.Target = duplicateTargetFromIntent(msg.Target)
	case intents.DuplicateOpenTargetPicker:
		r.targetPicker = target_picker.NewModel(r.context)
		return r.targetPicker.Init()
	case intents.Apply:
		target := targetToFlags[r.Target]
		return r.context.RunCommand(jj.Duplicate(r.From, r.targetArg(), target), common.RefreshAndSelect(r.From.Last()), common.Close)
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

func duplicateTargetFromIntent(target intents.DuplicateTarget) Target {
	switch target {
	case intents.DuplicateTargetDestination:
		return TargetDestination
	case intents.DuplicateTargetAfter:
		return TargetAfter
	case intents.DuplicateTargetBefore:
		return TargetBefore
	default:
		return TargetDestination
	}
}

func (r *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	r.To = commit
	return nil
}

func (r *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos == operations.RenderBeforeChangeId {
		if r.From.Contains(commit) {
			return r.styles.sourceMarker.Render("<< duplicate >>")
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
		r.styles.dimmed.Render(" duplicate "),
		r.styles.changeId.Render(strings.Join(r.From.GetIds(), " ")),
		r.styles.dimmed.Render("", ret, ""),
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
	return "duplicate"
}

func (r *Operation) Scope() keybindings.Scope {
	if r.targetPicker != nil {
		return keybindings.Scope(actions.OwnerTargetPicker)
	}
	return keybindings.Scope(actions.OwnerDuplicate)
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

func NewOperation(context *appContext.MainContext, from jj.SelectedRevisions, target Target) *Operation {
	styles := styles{
		changeId:     common.DefaultPalette.Get("duplicate change_id"),
		dimmed:       common.DefaultPalette.Get("duplicate dimmed"),
		sourceMarker: common.DefaultPalette.Get("duplicate source_marker"),
		targetMarker: common.DefaultPalette.Get("duplicate target_marker"),
	}
	return &Operation{
		context: context,
		From:    from,
		Target:  target,
		styles:  styles,
	}
}
