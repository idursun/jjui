package bookmark

import (
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveBookmarkOperation_PassesAllowBackwardsUnlessForward(t *testing.T) {
	for _, tc := range []struct {
		name, directions string
		label            string
		args             jj.CommandArgs
	}{
		{name: "forward", directions: "f\n", args: jj.BookmarkMove("target", []string{"main", "dev"})},
		{name: "backwards", directions: "f\nb\n", label: "backwards", args: jj.BookmarkMove("target", []string{"main", "dev"}, "--allow-backwards")},
		{name: "sideways", directions: "b\ns\n", label: "sideways", args: jj.BookmarkMove("target", []string{"main", "dev"}, "--allow-backwards")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			commandRunner := test.NewTestCommandRunner(t)
			commandRunner.Expect(jj.BookmarkMoveDirections("target", []string{"main", "dev"})).SetOutput([]byte(tc.directions))
			commandRunner.Expect(tc.args)
			defer commandRunner.Verify()

			target := &jj.Commit{ChangeId: "target"}
			op := NewMoveBookmarkOperation(test.NewTestContext(commandRunner), []string{"main", "dev"}, nil)
			test.SimulateModel(op, op.SetSelectedRevision(target))

			marker := test.Stripped(op.Render(target, operations.RenderPositionBefore))
			if tc.label == "" {
				assert.NotContains(t, marker, "backwards")
				assert.NotContains(t, marker, "sideways")
			} else {
				assert.Contains(t, marker, tc.label)
			}
			test.SimulateModel(op, op.Update(intents.Apply{}))
		})
	}
}

func TestMoveBookmarkOperation_ApplyAsksForADirectionNotYetKnown(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkMoveDirections("target", []string{"main"})).SetOutput([]byte("b\n"))
	commandRunner.Expect(jj.BookmarkMove("target", []string{"main"}, "--allow-backwards"))
	defer commandRunner.Verify()

	op := NewMoveBookmarkOperation(test.NewTestContext(commandRunner), []string{"main"}, nil)
	op.SetSelectedRevision(&jj.Commit{ChangeId: "target"})
	test.SimulateModel(op, op.Update(intents.Apply{}))
}

func TestMoveBookmarkOperation_ForceApplyRunsMoveWithAllowBackwards(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkMoveDirections("target", []string{"main"})).SetOutput([]byte("f\n"))
	commandRunner.Expect(jj.BookmarkMove("target", []string{"main"}, "--allow-backwards"))
	defer commandRunner.Verify()

	op := NewMoveBookmarkOperation(test.NewTestContext(commandRunner), []string{"main"}, nil)
	test.SimulateModel(op, op.SetSelectedRevision(&jj.Commit{ChangeId: "target"}))
	test.SimulateModel(op, op.Update(intents.Apply{Force: true}))
}

func TestMoveBookmarkOperation_CancelReturnsCommand(t *testing.T) {
	op := NewMoveBookmarkOperation(test.NewTestContext(test.NewTestCommandRunner(t)), []string{"main"}, nil)

	cmd := op.Update(intents.Cancel{})
	require.NotNil(t, cmd)
}

func TestMoveBookmarkOperation_IgnoresALateDirectionForAnEarlierTarget(t *testing.T) {
	op := NewMoveBookmarkOperation(test.NewTestContext(test.NewTestCommandRunner(t)), []string{"main"}, nil)
	op.SetSelectedRevision(&jj.Commit{ChangeId: "later"})
	op.Update(moveDirectionMsg{target: "later", direction: moveSideways})
	op.Update(moveDirectionMsg{target: "earlier", direction: moveForward})
	assert.Equal(t, moveSideways, op.direction)
	assert.Equal(t, "later", op.directionTarget)
}
