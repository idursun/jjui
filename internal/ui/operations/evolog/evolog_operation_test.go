package evolog

import (
	"testing"
	"time"

	"github.com/charmbracelet/x/exp/teatest"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
)

var revision = &jj.Commit{
	ChangeId:      "abc",
	IsWorkingCopy: false,
	Hidden:        false,
	CommitId:      "123",
}

func TestOperation_Init(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Evolog(revision.ChangeId))
	defer commandRunner.Verify()

	context := test.NewTestContext(commandRunner)
	operation := NewOperation(context, revision, 10, 20)
	tm := teatest.NewTestModel(t, operation)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.IsVerified()
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
