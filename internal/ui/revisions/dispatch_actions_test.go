package revisions

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestHandleDispatchedAction_RevisionsModeOpeners(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status("a"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.updateGraphRows(rows, "a")

	_, handled := model.HandleDispatchedAction(actions.RevisionsOpenDetails, nil)
	assert.True(t, handled, "open_details should be handled by revisions dispatcher")
	assert.Equal(t, "details", model.CurrentOperation().Name())

	model.Update(intents.Cancel{})
	model.updateGraphRows(rows, "a")

	_, handled = model.HandleDispatchedAction(actions.RevisionsOpenRebase, nil)
	assert.True(t, handled, "open_rebase should be handled by revisions dispatcher")
	assert.Equal(t, "rebase", model.CurrentOperation().Name())

	model.Update(intents.Cancel{})
	model.updateGraphRows(rows, "a")
	_, handled = model.HandleDispatchedAction(actions.RevisionsOpenDuplicate, nil)
	assert.True(t, handled, "open_duplicate should be handled by revisions dispatcher")
	assert.Equal(t, "duplicate", model.CurrentOperation().Name())
}

type confirmationTrackingOp struct {
	lastIntent  intents.OptionSelect
	gotNavigate bool
}

func (o *confirmationTrackingOp) ResolveAction(action keybindings.Action, args map[string]any) (intents.Intent, bool) {
	_ = args

	switch action {
	case actions.RevisionsDetailsConfirmationPrev:
		return intents.OptionSelect{Delta: -1}, true
	case actions.RevisionsDetailsConfirmationNext:
		return intents.OptionSelect{Delta: 1}, true
	case actions.RevisionsDetailsConfirmationApply:
		return intents.Apply{}, true
	case actions.RevisionsDetailsConfirmationCancel:
		return intents.Cancel{}, true
	default:
		return nil, false
	}
}

func (o *confirmationTrackingOp) Update(msg tea.Msg) tea.Cmd {
	intent, ok := msg.(intents.OptionSelect)
	if ok {
		o.gotNavigate = true
		o.lastIntent = intent
	}
	return nil
}

func (o *confirmationTrackingOp) Init() tea.Cmd                                   { return nil }
func (o *confirmationTrackingOp) ViewRect(_ *render.DisplayContext, _ layout.Box) {}
func (o *confirmationTrackingOp) IsFocused() bool                                 { return true }
func (o *confirmationTrackingOp) IsEditing() bool                                 { return true }
func (o *confirmationTrackingOp) Name() string                                    { return "details" }
func (o *confirmationTrackingOp) Render(_ *jj.Commit, _ operations.RenderPosition) string {
	return ""
}
func (o *confirmationTrackingOp) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}
func (o *confirmationTrackingOp) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
}

func TestHandleDispatchedAction_DetailsConfirmationMoveDoesNotMoveRevisionsCursor(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a")
	model.SetCursor(0)

	op := &confirmationTrackingOp{}
	model.op = op

	_, handled := model.HandleDispatchedAction(actions.RevisionsDetailsConfirmationNext, nil)
	assert.True(t, handled, "move_down should be handled in details confirmation scope")
	assert.True(t, op.gotNavigate, "details confirmation should receive DetailsConfirmationNavigate intent")
	assert.Equal(t, 1, op.lastIntent.Delta, "move_down should navigate confirmation to the right/down option")
	assert.Equal(t, 0, model.Cursor(), "revisions cursor must not move while details confirmation is active")
}

type applyTrackingOp struct {
	force bool
}

func (o *applyTrackingOp) ResolveAction(action keybindings.Action, args map[string]any) (intents.Intent, bool) {
	switch action {
	case actions.RevisionsSquashApply:
		return intents.Apply{Force: operations.BoolArg(args, "force", false)}, true
	default:
		return nil, false
	}
}

func (o *applyTrackingOp) Update(msg tea.Msg) tea.Cmd {
	if apply, ok := msg.(intents.Apply); ok {
		o.force = apply.Force
	}
	return nil
}

func (o *applyTrackingOp) Init() tea.Cmd                                   { return nil }
func (o *applyTrackingOp) ViewRect(_ *render.DisplayContext, _ layout.Box) {}
func (o *applyTrackingOp) IsFocused() bool                                 { return true }
func (o *applyTrackingOp) IsEditing() bool                                 { return true }
func (o *applyTrackingOp) Name() string                                    { return "squash" }
func (o *applyTrackingOp) Render(_ *jj.Commit, _ operations.RenderPosition) string {
	return ""
}
func (o *applyTrackingOp) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}
func (o *applyTrackingOp) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
}

func TestHandleDispatchedAction_ApplyArgsFlowToOperationResolver(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a")

	op := &applyTrackingOp{}
	model.op = op

	_, handled := model.HandleDispatchedAction(actions.RevisionsSquashApply, map[string]any{"force": true})
	assert.True(t, handled, "apply should be handled by operation resolver")
	assert.True(t, op.force, "force arg from binding should map to intents.Apply{Force:true}")
}

func TestHandleDispatchedAction_CancelClearsSelectionsInNormalMode(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a")

	rev := model.SelectedRevision()
	if assert.NotNil(t, rev, "expected a selected revision in test data") {
		ctx.AddCheckedItem(common.SelectedRevision{ChangeId: rev.GetChangeId(), CommitId: rev.CommitId})
	}
	before := len(ctx.CheckedItems)
	assert.Greater(t, before, 0, "setup should create at least one selected revision")

	_, handled := model.HandleDispatchedAction(actions.UiCancel, nil)
	assert.True(t, handled, "revisions cancel should be consumed in normal mode")
	assert.Equal(t, 0, len(ctx.CheckedItems), "cancel in normal mode should clear selected revisions")
}

func TestHandleDispatchedAction_CancelNoopWithoutSelectionsInNormalMode(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a")

	assert.Equal(t, 0, len(ctx.CheckedItems), "setup should start with no selected revisions")

	_, handled := model.HandleDispatchedAction(actions.UiCancel, nil)
	assert.True(t, handled, "revisions cancel should be consumed in normal mode")
	assert.Equal(t, 0, len(ctx.CheckedItems), "cancel should be a no-op when there are no selected revisions")
}
