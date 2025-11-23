package abandon

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
)

var commit = &jj.Commit{ChangeId: "a"}
var revisions = jj.NewSelectedRevisions(commit)

func Test_Accept(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Abandon(revisions, false))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), revisions)
	test.SimulateModel(model, model.Init())

	model.SetSelectedRevision(commit)
	test.SimulateModel(model, test.Press(tea.KeyEnter))
}

func Test_Cancel(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), revisions)
	test.SimulateModel(model, model.Init())

	model.SetSelectedRevision(commit)

	test.SimulateModel(model, test.Press(tea.KeyEsc))
}
