package new_between

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

func TestApplyUsesCurrentRevisionAsInsertBeforeWhenNothingIsPinned(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.NewInsert(
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "after-1"}, &jj.Commit{ChangeId: "after-2"}),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "current"}),
	))
	defer commandRunner.Verify()

	op := New(
		test.NewTestContext(commandRunner),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "after-1"}, &jj.Commit{ChangeId: "after-2"}),
		&jj.Commit{ChangeId: "current"},
	)

	cmd, handled := op.HandleIntent(intents.Apply{})
	require.True(t, handled)
	require.NotNil(t, cmd)

	var msgs []tea.Msg
	test.SimulateModel(op, cmd, func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	require.NotEmpty(t, msgs)
	_, closed := msgs[0].(common.CloseViewMsg)
	assert.True(t, closed, "operation should close before running new")
	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "after-1"}, operations.RenderBeforeChangeId), "<< after this >>")
	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "after-2"}, operations.RenderBeforeChangeId), "<< after this >>")
	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "current"}, operations.RenderBeforeChangeId), "<< before this >>")
}

func TestRenderShowsInsertAfterMarkerBeforeFallbackInsertBeforeMarker(t *testing.T) {
	op := New(
		test.NewTestContext(test.NewTestCommandRunner(t)),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "current"}),
		&jj.Commit{ChangeId: "current"},
	)

	marker := op.Render(&jj.Commit{ChangeId: "current"}, operations.RenderBeforeChangeId)
	assert.Contains(t, marker, "<< after this >>")
	assert.NotContains(t, marker, "<< before this >>")
}

func TestApplyUsesPlainNewWhenInsertAfterAndBeforeAreSameRevision(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.New(jj.NewSelectedRevisions(&jj.Commit{ChangeId: "current"})))
	defer commandRunner.Verify()

	op := New(
		test.NewTestContext(commandRunner),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "current"}),
		&jj.Commit{ChangeId: "current"},
	)

	cmd, handled := op.HandleIntent(intents.Apply{})
	require.True(t, handled)
	require.NotNil(t, cmd)

	test.SimulateModel(op, cmd)
}

func TestApplyUsesPinnedInsertBeforeRevisions(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.NewInsert(
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "after-1"}, &jj.Commit{ChangeId: "after-2"}),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "before-1"}, &jj.Commit{ChangeId: "before-2"}),
	))
	defer commandRunner.Verify()

	op := New(
		test.NewTestContext(commandRunner),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "after-1"}, &jj.Commit{ChangeId: "after-2"}),
		&jj.Commit{ChangeId: "before-1"},
	)

	cmd, handled := op.HandleIntent(intents.NewBetweenToggleInsertBefore{})
	require.True(t, handled)
	require.Nil(t, cmd)

	cmd = op.Update(common.SelectionChangedMsg{Item: common.SelectedRevision{ChangeId: "before-2"}})
	require.Nil(t, cmd)

	cmd, handled = op.HandleIntent(intents.NewBetweenToggleInsertBefore{})
	require.True(t, handled)
	require.Nil(t, cmd)

	cmd, handled = op.HandleIntent(intents.Apply{})
	require.True(t, handled)
	require.NotNil(t, cmd)

	test.SimulateModel(op, cmd)

	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "after-1"}, operations.RenderBeforeChangeId), "<< after this >>")
	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "after-2"}, operations.RenderBeforeChangeId), "<< after this >>")
	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "before-1"}, operations.RenderBeforeChangeId), "<< before this >>")
	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "before-2"}, operations.RenderBeforeChangeId), "<< before this >>")
}

func TestSelectionChangeUpdatesFallbackInsertBeforeWhenNothingIsPinned(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.NewInsert(
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "after"}),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "selected"}),
	))
	defer commandRunner.Verify()

	op := New(
		test.NewTestContext(commandRunner),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "after"}),
		&jj.Commit{ChangeId: "initial"},
	)

	cmd := op.Update(common.SelectionChangedMsg{
		Item: common.SelectedRevision{ChangeId: "selected", CommitId: "selected-commit"},
	})
	require.Nil(t, cmd)

	assert.NotContains(t, op.Render(&jj.Commit{ChangeId: "initial"}, operations.RenderBeforeChangeId), "<< before this >>")
	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "selected"}, operations.RenderBeforeChangeId), "<< before this >>")

	cmd, handled := op.HandleIntent(intents.Apply{})
	require.True(t, handled)
	require.NotNil(t, cmd)

	test.SimulateModel(op, cmd)
}

func TestPinnedInsertBeforeIgnoresSelectionChangesUntilPinsAreCleared(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.NewInsert(
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "after"}),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "pinned"}),
	))
	defer commandRunner.Verify()

	op := New(
		test.NewTestContext(commandRunner),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "after"}),
		&jj.Commit{ChangeId: "pinned", CommitId: "pinned-commit"},
	)

	cmd, handled := op.HandleIntent(intents.NewBetweenToggleInsertBefore{})
	require.True(t, handled)
	require.Nil(t, cmd)

	cmd = op.Update(common.SelectionChangedMsg{
		Item: common.SelectedRevision{ChangeId: "selected", CommitId: "selected-commit"},
	})
	require.Nil(t, cmd)

	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "pinned"}, operations.RenderBeforeChangeId), "<< before this >>")
	assert.NotContains(t, op.Render(&jj.Commit{ChangeId: "selected"}, operations.RenderBeforeChangeId), "<< before this >>")

	cmd, handled = op.HandleIntent(intents.Apply{})
	require.True(t, handled)
	require.NotNil(t, cmd)

	test.SimulateModel(op, cmd)
}

func TestTogglingPinnedRevisionAgainReturnsToCurrentSelectionFallback(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.NewInsert(
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "after"}),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "selected"}),
	))
	defer commandRunner.Verify()

	op := New(
		test.NewTestContext(commandRunner),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "after"}),
		&jj.Commit{ChangeId: "pinned", CommitId: "pinned-commit"},
	)

	cmd, handled := op.HandleIntent(intents.NewBetweenToggleInsertBefore{})
	require.True(t, handled)
	require.Nil(t, cmd)

	cmd, handled = op.HandleIntent(intents.NewBetweenToggleInsertBefore{})
	require.True(t, handled)
	require.Nil(t, cmd)

	cmd = op.Update(common.SelectionChangedMsg{
		Item: common.SelectedRevision{ChangeId: "selected", CommitId: "selected-commit"},
	})
	require.Nil(t, cmd)

	assert.NotContains(t, op.Render(&jj.Commit{ChangeId: "pinned"}, operations.RenderBeforeChangeId), "<< before this >>")
	assert.Contains(t, op.Render(&jj.Commit{ChangeId: "selected"}, operations.RenderBeforeChangeId), "<< before this >>")

	cmd, handled = op.HandleIntent(intents.Apply{})
	require.True(t, handled)
	require.NotNil(t, cmd)

	test.SimulateModel(op, cmd)
}

func TestApplyWithNoEndpointsIsHandledWithoutCommand(t *testing.T) {
	op := New(
		test.NewTestContext(test.NewTestCommandRunner(t)),
		jj.NewSelectedRevisions(),
		nil,
	)

	cmd, handled := op.HandleIntent(intents.Apply{})
	require.True(t, handled)
	assert.Nil(t, cmd)
}

func TestCancelClosesOperation(t *testing.T) {
	op := New(
		test.NewTestContext(test.NewTestCommandRunner(t)),
		jj.NewSelectedRevisions(),
		nil,
	)

	cmd, handled := op.HandleIntent(intents.Cancel{})
	require.True(t, handled)
	require.NotNil(t, cmd)

	msg := cmd()
	_, closed := msg.(common.CloseViewMsg)
	assert.True(t, closed)
}

func TestRenderIgnoresOtherPositions(t *testing.T) {
	op := New(
		test.NewTestContext(test.NewTestCommandRunner(t)),
		jj.NewSelectedRevisions(&jj.Commit{ChangeId: "after"}),
		&jj.Commit{ChangeId: "before"},
	)

	assert.Empty(t, op.Render(&jj.Commit{ChangeId: "after"}, operations.RenderPositionBefore))
	assert.Empty(t, op.Render(&jj.Commit{ChangeId: "before"}, operations.RenderPositionBefore))
}

func TestName(t *testing.T) {
	op := New(
		test.NewTestContext(test.NewTestCommandRunner(t)),
		jj.NewSelectedRevisions(),
		nil,
	)

	assert.Equal(t, "new.between", op.Name())
}
