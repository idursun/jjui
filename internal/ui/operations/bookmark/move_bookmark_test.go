package bookmark

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveBookmarkOperation_ApplyRunsMoveForMovableBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovable("target")).SetOutput([]byte("main;.;false;false;false;abc123\n"))
	commandRunner.Expect(jj.BookmarkMove("target", "main", "--allow-backwards"))
	defer commandRunner.Verify()

	op := NewMoveBookmarkOperation(test.NewTestContext(commandRunner), "main", nil)
	require.Nil(t, op.SetSelectedRevision(&jj.Commit{ChangeId: "target"}))

	cmd := op.Update(intents.Apply{})
	require.NotNil(t, cmd)
	test.SimulateModel(op, cmd)
}

func TestMoveBookmarkOperation_ApplyShowsMessageForNonMovableBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovable("target")).SetOutput([]byte("other;.;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	op := NewMoveBookmarkOperation(test.NewTestContext(commandRunner), "main", nil)
	require.Nil(t, op.SetSelectedRevision(&jj.Commit{ChangeId: "target"}))

	cmd := op.Update(intents.Apply{})
	require.NotNil(t, cmd)
	msg, ok := cmd().(intents.AddMessage)
	require.True(t, ok)
	assert.Equal(t, "Bookmark main can't be moved to target", msg.Text)
}

func TestMoveBookmarkOperation_CancelClosesThenRunsExit(t *testing.T) {
	exitCalled := false
	op := NewMoveBookmarkOperation(test.NewTestContext(test.NewTestCommandRunner(t)), "main", func() tea.Msg {
		exitCalled = true
		return nil
	})

	cmd := op.Update(intents.Cancel{})
	require.NotNil(t, cmd)
	test.SimulateModel(op, cmd)
	assert.True(t, exitCalled)
}
