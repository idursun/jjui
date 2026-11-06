package bookmark

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestRenameBookmarkOperation_RenamesToEditedName(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkRename("old", "old-new"))
	defer commandRunner.Verify()

	op := NewRenameBookmarkOperation(test.NewTestContext(commandRunner), "revision", "old")
	assert.Equal(t, "old", op.name.Value(), "the input starts with the current name")
	test.SimulateModel(op, test.Type(" new"))
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func TestRenameBookmarkOperation_UnchangedNameRunsNothing(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	op := NewRenameBookmarkOperation(test.NewTestContext(commandRunner), "revision", "old")
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}
