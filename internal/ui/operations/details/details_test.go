package details

import (
	"bytes"
	"testing"
	"time"

	"github.com/idursun/jjui/internal/jj"

	"github.com/idursun/jjui/test"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

const (
	Revision     = "ignored"
	StatusOutput = "false false\nM file.txt\nA newfile.txt\n"
)

var Commit = &jj.Commit{
	ChangeId: Revision,
	CommitId: Revision,
}

func TestModel_Init_ExecutesStatusCommand(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit, 10)
	tm := teatest.NewTestModel(t, model)
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

	tm := teatest.NewTestModel(t, NewOperation(test.NewTestContext(commandRunner), Commit, 10))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.IsVerified()
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestModel_Update_SplitsSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Split(Revision, []string{"file.txt"}, false))
	defer commandRunner.Verify()

	tm := teatest.NewTestModel(t, NewOperation(test.NewTestContext(commandRunner), Commit, 10))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.IsVerified()
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestModel_Update_ParallelSplitsSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Split(Revision, []string{"file.txt"}, true))
	defer commandRunner.Verify()

	tm := teatest.NewTestModel(t, NewOperation(test.NewTestContext(commandRunner), Commit, 10))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s"), Alt: true})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.IsVerified()
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestModel_Update_HandlesMovedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false\nR internal/ui/{revisions => }/file.go\nR {file => sub/newfile}\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"internal/ui/file.go", "sub/newfile"}))
	defer commandRunner.Verify()

	tm := teatest.NewTestModel(t, NewOperation(test.NewTestContext(commandRunner), Commit, 10))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.go"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.IsVerified()
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestModel_Update_HandlesMovedFilesInDeepDirectories(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false false\nR {src/new_file_3.md => new_file.md}\nR src/{new_file.py => renamed_py.py}\nR {src1/to_be_renamed.md => src2/renamed.md}\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"new_file.md", "src/renamed_py.py", "src2/renamed.md"}))
	defer commandRunner.Verify()

	tm := teatest.NewTestModel(t, NewOperation(test.NewTestContext(commandRunner), Commit, 10))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("new_file.md"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.IsVerified()
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestModel_Update_HandlesFilenamesWithBraces(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false\nM file{with}braces.txt\nA another{test}.go\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"file{with}braces.txt", "another{test}.go"}))
	defer commandRunner.Verify()

	tm := teatest.NewTestModel(t, NewOperation(test.NewTestContext(commandRunner), Commit, 10))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file{with}braces.txt"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.IsVerified()
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
