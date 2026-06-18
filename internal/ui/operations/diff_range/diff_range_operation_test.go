package diff_range

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyClosesOperationBeforeShowingDiff(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.DiffRange("from", "to")).
		SetOutput([]byte("diff content"))
	defer commandRunner.Verify()

	op := New(test.NewTestContext(commandRunner), &jj.Commit{ChangeId: "from"}, &jj.Commit{ChangeId: "to"})

	cmd, handled := op.HandleIntent(intents.Apply{})
	require.True(t, handled)
	require.NotNil(t, cmd)

	var msgs []tea.Msg
	test.SimulateModel(op, cmd, func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	require.Len(t, msgs, 2)
	_, closed := msgs[0].(common.CloseViewMsg)
	assert.True(t, closed, "operation should close before diff view opens")

	diff, shown := msgs[1].(intents.DiffShow)
	require.True(t, shown)
	assert.Equal(t, "diff content", diff.Content)
}

func TestSwapReversesDiffEndpoints(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.DiffRange("to", "from")).
		SetOutput([]byte("reversed diff content"))
	defer commandRunner.Verify()

	op := New(test.NewTestContext(commandRunner), &jj.Commit{ChangeId: "from"}, &jj.Commit{ChangeId: "to"})

	cmd, handled := op.HandleIntent(intents.DiffRangeSwap{})
	require.True(t, handled)
	require.Nil(t, cmd)

	cmd, handled = op.HandleIntent(intents.Apply{})
	require.True(t, handled)
	require.NotNil(t, cmd)

	var msgs []tea.Msg
	test.SimulateModel(op, cmd, func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	require.Len(t, msgs, 2)
	diff, shown := msgs[1].(intents.DiffShow)
	require.True(t, shown)
	assert.Equal(t, "reversed diff content", diff.Content)
}

func TestSwapKeepsTargetMarkerAfterSelectionSync(t *testing.T) {
	oldTarget := &jj.Commit{ChangeId: "to"}
	op := New(test.NewTestContext(test.NewTestCommandRunner(t)), &jj.Commit{ChangeId: "from"}, oldTarget)

	cmd, handled := op.HandleIntent(intents.DiffRangeSwap{})
	require.True(t, handled)
	require.Nil(t, cmd)

	selectRevision(op, oldTarget)

	assert.Contains(t, op.Render(oldTarget, operations.RenderPositionBefore), "<< from >>")
	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "from"}, operations.RenderBeforeChangeId), "<< to >>")

	selectRevision(op, &jj.Commit{ChangeId: "next"})
	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "next"}, operations.RenderPositionBefore), "<< from >>")
	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "from"}, operations.RenderBeforeChangeId), "<< to >>")
}

func selectRevision(op *Operation, commit *jj.Commit) {
	op.Update(common.SelectionChangedMsg{
		Item: common.SelectedRevision{
			ChangeId: commit.ChangeId,
			CommitId: commit.CommitId,
		},
	})
}
