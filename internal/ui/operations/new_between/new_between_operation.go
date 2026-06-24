package new_between

import (
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
)

var _ operations.Operation = (*Operation)(nil)
var _ common.Focusable = (*Operation)(nil)
var _ common.ScopeProvider = (*Operation)(nil)
var _ common.ScopeHandler = (*Operation)(nil)

type Operation struct {
	context      *appContext.MainContext
	insertAfter  jj.SelectedRevisions
	insertBefore jj.SelectedRevisions
	current      *jj.Commit
}

func (o *Operation) IsFocused() bool {
	return true
}

func (o *Operation) Scopes() []common.Scope {
	return []common.Scope{
		{
			Name:    actions.ScopeNewBetween,
			Leak:    common.LeakAll,
			Handler: o,
		},
	}
}

func (o *Operation) Init() tea.Cmd {
	return nil
}

func (o *Operation) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case common.SelectionChangedMsg:
		selected, ok := msg.Item.(common.SelectedRevision)
		if !ok {
			return nil
		}
		o.current = &jj.Commit{ChangeId: selected.ChangeId, CommitId: selected.CommitId}
		return nil
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
		insertBefore := o.effectiveInsertBefore()
		if len(o.insertAfter.Revisions) == 0 && len(insertBefore.Revisions) == 0 {
			return nil, true
		}
		return tea.Sequence(
			common.Close,
			o.context.RunCommand(o.newCommand(insertBefore), common.RefreshAndSelect("@")),
		), true
	case intents.NewBetweenToggleInsertBefore:
		o.insertBefore = o.insertBefore.Toggle(o.current)
		return nil, true
	}
	return nil, false
}

func (o *Operation) ViewRect(dl *render.DisplayContext, box layout.Box) {}

func (o *Operation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos != operations.RenderBeforeChangeId {
		return ""
	}

	sourceMarkerStyle := common.DefaultPalette.Get("new source_marker")
	targetMarkerStyle := common.DefaultPalette.Get("new target_marker")
	isInsertAfter := o.insertAfter.Contains(commit)
	isInsertBefore := false
	if len(o.insertBefore.Revisions) > 0 {
		isInsertBefore = o.insertBefore.Contains(commit)
	} else {
		isInsertBefore = o.current != nil && o.current.GetChangeId() == commit.GetChangeId()
	}

	if isInsertAfter {
		return sourceMarkerStyle.Render("<< after this >>")
	}
	if isInsertBefore {
		return targetMarkerStyle.Render("<< before this >>")
	}
	return ""
}

func (o *Operation) Name() string {
	return "new.between"
}

func New(context *appContext.MainContext, insertAfter jj.SelectedRevisions, current *jj.Commit) *Operation {
	return &Operation{context: context, insertAfter: insertAfter, current: current}
}

func (o *Operation) effectiveInsertBefore() jj.SelectedRevisions {
	if len(o.insertBefore.Revisions) > 0 {
		return o.insertBefore
	}
	return jj.NewSelectedRevisions(o.current)
}

func (o *Operation) newCommand(insertBefore jj.SelectedRevisions) jj.CommandArgs {
	// jj rejects inserting both after and before the same commit; fall back to a normal child commit.
	if len(o.insertAfter.Revisions) == 1 && len(insertBefore.Revisions) == 1 && o.insertAfter.Revisions[0].Equal(insertBefore.Revisions[0]) {
		return jj.New(o.insertAfter)
	}
	return jj.NewInsert(o.insertAfter, insertBefore)
}
