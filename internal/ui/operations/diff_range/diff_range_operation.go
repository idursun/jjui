package diff_range

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/target_picker"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ operations.Operation = (*Operation)(nil)
var _ common.Focusable = (*Operation)(nil)
var _ common.ScopeProvider = (*Operation)(nil)
var _ common.ScopeHandler = (*Operation)(nil)

type Operation struct {
	context *appContext.MainContext
	from    *jj.Commit
	to      *jj.Commit
	toName  string
	swapped bool
}

func (o *Operation) IsFocused() bool {
	return true
}

func (o *Operation) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeDiffRange,
			Leak:    common.LeakAll,
			Handler: o,
		},
	}
}

func (o *Operation) setSelectedRevision(commit *jj.Commit) tea.Cmd {
	if o.swapped {
		if o.from.Equal(commit) {
			return nil
		}
		o.from = commit
	} else {
		if o.to.Equal(commit) {
			return nil
		}
		o.to = commit
	}
	return nil
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case target_picker.TargetSelectedMsg:
		o.toName = strings.TrimSpace(msg.Target)
		cmd, _ := o.HandleIntent(intents.Apply{})
		return cmd
	case common.SelectionChangedMsg:
		selected, ok := msg.Item.(common.SelectedRevision)
		if !ok {
			return nil
		}
		return o.setSelectedRevision(&jj.Commit{ChangeId: selected.ChangeId, CommitId: selected.CommitId})
	case intents.Intent:
		cmd, _ := o.HandleIntent(msg)
		return cmd
	default:
		return nil
	}
}

func (o *Operation) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.Cancel:
		return common.Close, true
	case intents.Apply:
		command := func() tea.Msg {
			if output, err := o.context.RunCommandImmediate(jj.DiffRange(o.from.GetChangeId(), o.targetArg())); err != nil {
				return intents.AddMessage{Text: err.Error()}
			} else {
				return intents.DiffShow{Content: string(output)}
			}
		}
		return tea.Sequence(common.Close, command), true
	case intents.DiffRangeOpenTargetPicker:
		return common.OpenTargetPicker(), true
	case intents.DiffRangeSwap:
		if o.from == nil || o.to == nil {
			return nil, true
		}
		o.from, o.to = o.to, o.from
		o.swapped = !o.swapped
		return nil, true
	}
	return nil, false
}

func (o *Operation) ViewRect(dl *render.DisplayContext, box layout.Box) {}

func (o *Operation) Render(commit *jj.Commit, renderPosition operations.RenderPosition) string {
	if renderPosition == operations.RenderPositionBefore && o.from != nil && commit.GetChangeId() == o.from.GetChangeId() {
		sourceMarkerStyle := common.DefaultPalette.Get("diff_range source_marker")
		dimmedStyle := common.DefaultPalette.Get("diff_range dimmed")
		return lipgloss.JoinHorizontal(0, sourceMarkerStyle.Render("<< from >>"), dimmedStyle.Render(" excluding this revision"))
	}

	if renderPosition == operations.RenderPositionBefore && o.to != nil && commit.GetChangeId() == o.to.GetChangeId() {
		targetMarkerStyle := common.DefaultPalette.Get("diff_range target_marker").PaddingRight(1)
		changeIdStyle := common.DefaultPalette.Get("diff_range change_id")
		dimmedStyle := common.DefaultPalette.Get("diff_range dimmed")
		commandHint := lipgloss.JoinHorizontal(0, dimmedStyle.Render(" jj diff --from "), changeIdStyle.Render(o.from.GetChangeId()), dimmedStyle.Render(" --to "), changeIdStyle.Render(o.to.GetChangeId()))
		return lipgloss.JoinHorizontal(0, targetMarkerStyle.Render("<< to >>"), commandHint)
	}
	return ""
}

func (o *Operation) targetArg() string {
	if strings.TrimSpace(o.toName) != "" {
		return o.toName
	}
	return o.to.GetChangeId()
}

func (o *Operation) Name() string {
	return "diff.range"
}

func New(context *appContext.MainContext, source *jj.Commit, current *jj.Commit) *Operation {
	return &Operation{context: context, from: source, to: current}
}
