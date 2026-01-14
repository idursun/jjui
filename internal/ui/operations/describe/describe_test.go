package describe

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestOperation_Update_RemembersDiscardedDescription(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change_id")).SetOutput([]byte(""))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	operation := NewOperation(ctx, &jj.Commit{ChangeId: "change_id"})
	test.SimulateModel(operation, operation.Init())
	test.SimulateModel(operation, test.Type("Some description"))
	test.SimulateModel(operation, test.Press(tea.KeyEsc))
	assert.Equal(t, "Some description", stashed.description)
}

func TestOperation_Update_RestoresStashedDescription(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change_id")).SetOutput([]byte(""))
	revision := &jj.Commit{ChangeId: "change_id", CommitId: "commit_id"}
	defer commandRunner.Verify()

	stashed = &stashedDescription{
		revision:    revision,
		description: "restored description",
	}
	defer func() {
		stashed = nil
	}()

	ctx := test.NewTestContext(commandRunner)
	operation := NewOperation(ctx, revision)
	test.SimulateModel(operation, operation.Init())
	view := test.RenderImmediate(operation, 100, 20)
	assert.Contains(t, view, "restored description")
}
