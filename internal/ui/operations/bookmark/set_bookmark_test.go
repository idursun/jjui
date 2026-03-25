package bookmark

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestSetBookmarkModel_Update(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovable("revision"))
	commandRunner.Expect(jj.BookmarkSet("revision", "name"))
	defer commandRunner.Verify()

	op := NewSetBookmarkOperation(test.NewTestContext(commandRunner), "revision")
	test.SimulateModel(op, op.Init())
	test.SimulateModel(op, test.Type("name"))
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func TestSetBookmarkModel_WithReturnFocus_EmitsFocusBookmarkViewOnApply(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovable("revision"))
	commandRunner.Expect(jj.BookmarkSet("revision", "name"))
	defer commandRunner.Verify()

	op := NewSetBookmarkOperationWithReturnFocus(test.NewTestContext(commandRunner), "revision")
	test.SimulateModel(op, op.Init())
	test.SimulateModel(op, test.Type("name"))

	focused := false
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} }, func(msg tea.Msg) {
		if _, ok := msg.(common.FocusBookmarkViewMsg); ok {
			focused = true
		}
	})

	assert.True(t, focused)
}
