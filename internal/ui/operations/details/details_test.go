package details

import (
	"bytes"
	"github.com/idursun/jjui/internal/jj"
	"testing"
	"time"

	"github.com/idursun/jjui/test"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

const (
	Revision     = "ignored"
	StatusOutput = "false false\nM file.txt\nA newfile.txt\n"
)

func TestModel_Init_ExecutesStatusCommand(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	defer commandRunner.Verify()

	tm := teatest.NewTestModel(t, New(test.NewTestContext(commandRunner), Revision))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})
}

func TestModel_Update_RestoresSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Restore(Revision, []string{"file.txt"}))
	defer commandRunner.Verify()

	tm := teatest.NewTestModel(t, test.NewShell(New(test.NewTestContext(commandRunner), Revision)))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestModel_Update_SplitsSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Split(Revision, []string{"file.txt"}))
	defer commandRunner.Verify()

	tm := teatest.NewTestModel(t, test.NewShell(New(test.NewTestContext(commandRunner), Revision)))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestModel_Update_HandlesMovedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false\nR internal/ui/{revisions => }/file.go\nR {file => sub/newfile}\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"internal/ui/file.go", "sub/newfile"}))
	defer commandRunner.Verify()

	tm := teatest.NewTestModel(t, test.NewShell(New(test.NewTestContext(commandRunner), Revision)))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.go"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
