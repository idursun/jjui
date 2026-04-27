package bookmark

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
)

func TestSetBookmarkModel_Update(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovable("revision"))
	commandRunner.Expect(jj.BookmarkSet("revision", "name"))
	defer commandRunner.Verify()

	op := NewSetBookmarkOperation(test.NewTestContext(commandRunner), "revision", "")
	test.SimulateModel(op, op.Init())
	test.SimulateModel(op, test.Type("name"))
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func TestSetBookmarkModel_Prefill(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovable("revision"))
	commandRunner.Expect(jj.BookmarkSet("revision", "rdeaton/20260425/feature"))
	defer commandRunner.Verify()

	op := NewSetBookmarkOperation(test.NewTestContext(commandRunner), "revision", "rdeaton/20260425/")
	test.SimulateModel(op, op.Init())
	if got := op.name.Value(); got != "rdeaton/20260425/" {
		t.Fatalf("expected prefilled value %q, got %q", "rdeaton/20260425/", got)
	}
	test.SimulateModel(op, test.Type("feature"))
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}
