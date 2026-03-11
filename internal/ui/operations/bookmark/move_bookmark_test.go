package bookmark

import (
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/require"
)

func TestMoveBookmarkOperation_ApplyRunsMove(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkMove("target", "main"))
	defer commandRunner.Verify()

	op := NewMoveBookmarkOperation(test.NewTestContext(commandRunner), "main")
	require.Nil(t, op.SetSelectedRevision(&jj.Commit{ChangeId: "target"}))

	cmd := op.Update(intents.Apply{})
	require.NotNil(t, cmd)
	test.SimulateModel(op, cmd)
}

func TestMoveBookmarkOperation_ForceApplyRunsMoveWithAllowBackwards(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkMove("target", "main", "--allow-backwards"))
	defer commandRunner.Verify()

	op := NewMoveBookmarkOperation(test.NewTestContext(commandRunner), "main")
	require.Nil(t, op.SetSelectedRevision(&jj.Commit{ChangeId: "target"}))

	cmd := op.Update(intents.Apply{Force: true})
	require.NotNil(t, cmd)
	test.SimulateModel(op, cmd)
}

func TestMoveBookmarkOperation_CancelReturnsCommand(t *testing.T) {
	op := NewMoveBookmarkOperation(test.NewTestContext(test.NewTestCommandRunner(t)), "main")

	cmd := op.Update(intents.Cancel{})
	require.NotNil(t, cmd)
}
