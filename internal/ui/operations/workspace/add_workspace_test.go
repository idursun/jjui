package workspace

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
)

func TestAddWorkspaceOperation_Apply(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.WorkspaceAdd("revision", "../my ws"))
	defer commandRunner.Verify()

	op := NewAddWorkspaceOperation(test.NewTestContext(commandRunner), "revision", "../")
	test.SimulateModel(op, op.Init())
	test.SimulateModel(op, test.Type("my ws"))
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}
