package duplicate

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/dispatch"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/target_picker"
	"github.com/idursun/jjui/internal/ui/render"
)

var (
	targetToFlags = map[intents.ModeTarget]string{
		intents.ModeTargetAfter:       "--insert-after",
		intents.ModeTargetBefore:      "--insert-before",
		intents.ModeTargetDestination: "--onto",
	}
)

var _ operations.Operation = (*Operation)(nil)
var _ common.Focusable = (*Operation)(nil)
var _ dispatch.ScopeProvider = (*Operation)(nil)

type Operation struct {
	context     *appContext.MainContext
	From        jj.SelectedRevisions
	InsertStart *jj.Commit
	To          *jj.Commit
	Target      intents.ModeTarget
	targetName  string
}

func (r *Operation) IsFocused() bool {
	return true
}

func (r *Operation) Scopes() []dispatch.Scope {
	return []dispatch.Scope{
		{
			Name:    actions.ScopeDuplicate,
			Leak:    dispatch.LeakAll,
			Handler: r,
		},
	}
}

func (r *Operation) Init() tea.Cmd {
	return nil
}

func (r *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case target_picker.TargetSelectedMsg:
		r.targetName = strings.TrimSpace(msg.Target)
		cmd, _ := r.HandleIntent(intents.Apply{Force: msg.Force})
		return cmd
	case intents.Intent:
		cmd, _ := r.HandleIntent(msg)
		return cmd
	}
	return nil
}

func (r *Operation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent := intent.(type) {
	case intents.StartAceJump:
		return common.StartAceJump(), true
	case intents.DuplicateSetTarget:
		r.Target = intent.Target
		if r.Target == intents.ModeTargetInsert {
			r.InsertStart = r.To
		}
		return nil, true
	case intents.DuplicateOpenTargetPicker:
		return common.OpenTargetPicker(), true
	case intents.Apply:
		if r.Target == intents.ModeTargetInsert {
			insertAfter := r.InsertStart.GetChangeId()
			insertBefore := r.targetArg()
			return r.context.RunCommand(jj.DuplicateInsert(r.From, insertAfter, insertBefore), common.RefreshAndSelect(r.From.Last()), common.CloseApplied), true
		}
		target := targetToFlags[r.Target]
		return r.context.RunCommand(jj.Duplicate(r.From, r.targetArg(), target), common.RefreshAndSelect(r.From.Last()), common.CloseApplied), true
	case intents.Cancel:
		return common.Close, true
	}
	return nil, false
}

func (r *Operation) SetSelectedRevision(commit *jj.Commit) tea.Cmd {
	r.To = commit
	return nil
}

func (r *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	changeIdStyle := common.DefaultPalette.Get("duplicate change_id")
	dimmedStyle := common.DefaultPalette.Get("duplicate dimmed")
	sourceMarker := common.DefaultPalette.Get("duplicate source_marker")
	targetMarkerStyle := common.DefaultPalette.Get("duplicate target_marker")

	if pos == operations.RenderBeforeChangeId {
		changeId := commit.GetChangeId()
		if r.From.Contains(commit) {
			return sourceMarker.Render("<< duplicate >>")
		}
		if r.Target == intents.ModeTargetInsert && r.InsertStart != nil && r.InsertStart.GetChangeId() == changeId {
			return sourceMarker.Render("<< after this >>")
		}
		if r.Target == intents.ModeTargetInsert && r.To != nil && r.To.GetChangeId() == changeId {
			return sourceMarker.Render("<< before this >>")
		}
		return ""
	}
	expectedPos := operations.RenderPositionBefore
	if r.Target == intents.ModeTargetBefore || r.Target == intents.ModeTargetInsert {
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
	if r.Target == intents.ModeTargetDestination {
		ret = "onto"
	}
	if r.Target == intents.ModeTargetAfter {
		ret = "after"
	}
	if r.Target == intents.ModeTargetBefore {
		ret = "before"
	}
	if r.Target == intents.ModeTargetInsert {
		ret = "insert"
	}

	if r.Target == intents.ModeTargetInsert {
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			targetMarkerStyle.Render("<< insert >>"),
			" ",
			dimmedStyle.Render("duplicate "),
			changeIdStyle.Render(strings.Join(r.From.GetIds(), " ")),
			dimmedStyle.Render(" between "),
			changeIdStyle.Render(r.InsertStart.GetChangeId()),
			dimmedStyle.Render(" and "),
			changeIdStyle.Render(r.To.GetChangeId()),
		)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		targetMarkerStyle.Render("<< "+ret+" >>"),
		dimmedStyle.Render(" duplicate "),
		changeIdStyle.Render(strings.Join(r.From.GetIds(), " ")),
		dimmedStyle.Render("", ret, ""),
		changeIdStyle.Render(r.To.GetChangeId()),
	)
}

func (r *Operation) Name() string {
	return "duplicate"
}

func (r *Operation) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (r *Operation) targetArg() string {
	if strings.TrimSpace(r.targetName) != "" {
		return r.targetName
	}
	if r.To != nil {
		return r.To.GetChangeId()
	}
	return ""
}

func NewOperation(context *appContext.MainContext, from jj.SelectedRevisions, target intents.ModeTarget) *Operation {
	return &Operation{
		context: context,
		From:    from,
		Target:  target,
	}
}
