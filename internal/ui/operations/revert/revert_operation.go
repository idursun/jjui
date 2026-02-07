package revert

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
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
	TargetInsert
)

var (
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

var _ operations.Operation = (*Operation)(nil)
var _ common.Focusable = (*Operation)(nil)
var _ common.Overlay = (*Operation)(nil)
var _ common.Editable = (*Operation)(nil)

type Operation struct {
	context        *context.MainContext
	From           jj.SelectedRevisions
	InsertStart    *jj.Commit
	To             *jj.Commit
	Target         Target
	targetName     string
	targetPicker   *target_picker.Model
	keyMap         config.KeyMappings[key.Binding]
	highlightedIds []string
	styles         styles
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
		return r.handleIntent(msg)
	case tea.KeyMsg:
		if r.targetPicker != nil {
			return r.targetPicker.Update(msg)
		}
		return r.HandleKey(msg)
	default:
		if r.targetPicker != nil {
			return r.targetPicker.Update(msg)
		}
		return nil
	}
}

func (r *Operation) handleIntent(intent intents.Intent) tea.Cmd {
	switch msg := intent.(type) {
	case intents.RevertSetTarget:
		r.Target = revertTargetFromIntent(msg.Target)
		if r.Target == TargetInsert {
			r.InsertStart = r.To
		}
	case intents.Apply:
		if r.Target == TargetInsert {
			insertAfter := r.InsertStart.GetChangeId()
			insertBefore := r.targetArg()
			return r.context.RunCommand(jj.RevertInsert(r.From, insertAfter, insertBefore), common.RefreshAndSelect(r.From.Last()), common.Close)
		}
		source := "--revisions"
		target := targetToFlags[r.Target]
		return r.context.RunCommand(jj.Revert(r.From, r.targetArg(), source, target), common.RefreshAndSelect(r.From.Last()), common.Close)
	case intents.Cancel:
		return common.Close
	default:
		return nil
	}
	return nil
}

func revertTargetFromIntent(target intents.RevertTarget) Target {
	switch target {
	case intents.RevertTargetDestination:
		return TargetDestination
	case intents.RevertTargetAfter:
		return TargetAfter
	case intents.RevertTargetBefore:
		return TargetBefore
	case intents.RevertTargetInsert:
		return TargetInsert
	default:
		return TargetDestination
	}
}

func (r *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	if r.targetPicker != nil {
		return r.targetPicker.Update(msg)
	}
	switch {
	case key.Matches(msg, r.keyMap.Revert.Onto):
		return r.handleIntent(intents.RevertSetTarget{Target: intents.RevertTargetDestination})
	case key.Matches(msg, r.keyMap.Revert.After):
		return r.handleIntent(intents.RevertSetTarget{Target: intents.RevertTargetAfter})
	case key.Matches(msg, r.keyMap.Revert.Before):
		return r.handleIntent(intents.RevertSetTarget{Target: intents.RevertTargetBefore})
	case key.Matches(msg, r.keyMap.Revert.Insert):
		return r.handleIntent(intents.RevertSetTarget{Target: intents.RevertTargetInsert})
	case key.Matches(msg, r.keyMap.Revert.Target):
		r.targetPicker = target_picker.NewModel(r.context)
		return r.targetPicker.Init()
	case key.Matches(msg, r.keyMap.Apply):
		return r.handleIntent(intents.Apply{})
	case key.Matches(msg, r.keyMap.Cancel):
		return r.handleIntent(intents.Cancel{})
	}
	return nil
}

func (r *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	r.highlightedIds = nil
	r.To = commit
	r.highlightedIds = r.From.GetIds()
	return nil
}

func (r *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		r.keyMap.Apply,
		r.keyMap.Cancel,
		r.keyMap.Revert.Before,
		r.keyMap.Revert.After,
		r.keyMap.Revert.Onto,
		r.keyMap.Revert.Insert,
		r.keyMap.Revert.Target,
	}
}

func (r *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{r.ShortHelp()}
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

func (r *Operation) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}

func (r *Operation) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
}

func (r *Operation) Name() string {
	return "revert"
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
		keyMap:  config.Current.GetKeyMap(),
		From:    from,
		Target:  target,
		styles:  styles,
	}
}
