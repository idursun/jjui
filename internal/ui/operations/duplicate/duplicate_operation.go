package duplicate

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
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

type Operation struct {
	context     *appContext.MainContext
	From        jj.SelectedRevisions
	InsertStart *jj.Commit
	To          *jj.Commit
	Target      Target
	keyMap      config.KeyMappings[key.Binding]
	styles      styles
}

func (r *Operation) IsFocused() bool {
	return true
}

func (r *Operation) Init() tea.Cmd {
	return nil
}

func (r *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		return r.handleIntent(msg)
	case tea.KeyMsg:
		return r.HandleKey(msg)
	default:
		return nil
	}
}

func (r *Operation) handleIntent(intent intents.Intent) tea.Cmd {
	switch msg := intent.(type) {
	case intents.StartAceJump:
		return common.StartAceJump()
	case intents.DuplicateSetTarget:
		r.Target = duplicateTargetFromIntent(msg.Target)
	case intents.Apply:
		target := targetToFlags[r.Target]
		return r.context.RunCommand(jj.Duplicate(r.From, r.To.GetChangeId(), target), common.RefreshAndSelect(r.From.Last()), common.Close)
	case intents.Cancel:
		return common.Close
	default:
		return nil
	}
	return nil
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

func (r *Operation) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (r *Operation) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, r.keyMap.AceJump):
		return r.handleIntent(intents.StartAceJump{})
	case key.Matches(msg, r.keyMap.Duplicate.Onto):
		return r.handleIntent(intents.DuplicateSetTarget{Target: intents.DuplicateTargetDestination})
	case key.Matches(msg, r.keyMap.Duplicate.After):
		return r.handleIntent(intents.DuplicateSetTarget{Target: intents.DuplicateTargetAfter})
	case key.Matches(msg, r.keyMap.Duplicate.Before):
		return r.handleIntent(intents.DuplicateSetTarget{Target: intents.DuplicateTargetBefore})
	case key.Matches(msg, r.keyMap.Apply):
		return r.handleIntent(intents.Apply{})
	case key.Matches(msg, r.keyMap.Cancel):
		return r.handleIntent(intents.Cancel{})
	}
	return nil
}

func (r *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	r.To = commit
	return nil
}

func (r *Operation) ShortHelp() []key.Binding {
	return []key.Binding{
		r.keyMap.Cancel,
		r.keyMap.Duplicate.After,
		r.keyMap.Duplicate.Before,
		r.keyMap.Duplicate.Onto,
		r.keyMap.AceJump,
	}
}

func (r *Operation) FullHelp() [][]key.Binding {
	return [][]key.Binding{r.ShortHelp()}
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

func NewOperation(context *appContext.MainContext, from jj.SelectedRevisions, target Target) *Operation {
	styles := styles{
		changeId:     common.DefaultPalette.Get("duplicate change_id"),
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
