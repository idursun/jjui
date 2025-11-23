package details

import (
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/stretchr/testify/assert"

	"github.com/idursun/jjui/test"

	tea "github.com/charmbracelet/bubbletea"
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
	test.SimulateModel(model, model.Init())
	assert.Contains(t, model.View(), "file.txt")
}

func TestModel_Update_RestoresSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Restore(Revision, []string{"file.txt"}))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit, 10)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, model.View(), "file.txt")

	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Type("r"))
	test.SimulateModel(model, test.Press(tea.KeyEnter))
}

func TestModel_Update_RestoresInteractively(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.RestoreInteractive(Revision, "file.txt"))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit, 10)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, model.View(), "file.txt")
	test.SimulateModel(model, test.Type("ri"))
}

func TestModel_Update_SplitsSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Split(Revision, []string{"file.txt"}, false))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit, 10)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, model.View(), "file.txt")

	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Type("s"))
	test.SimulateModel(model, test.Press(tea.KeyEnter))
}

func TestModel_Update_ParallelSplitsSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Split(Revision, []string{"file.txt"}, true))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit, 10)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, model.View(), "file.txt")

	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, func() tea.Msg {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s"), Alt: true}
	})
	test.SimulateModel(model, test.Press(tea.KeyEnter))
}

func TestModel_Update_HandlesMovedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false\nR internal/ui/{revisions => }/file.go\nR {file => sub/newfile}\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"internal/ui/file.go", "sub/newfile"}))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit, 10)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, model.View(), "file.go")

	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Type("r"))
	test.SimulateModel(model, test.Press(tea.KeyEnter))
}

func TestModel_Update_HandlesMovedFilesInDeepDirectories(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false false\nR {src/new_file_3.md => new_file.md}\nR src/{new_file.py => renamed_py.py}\nR {src1/to_be_renamed.md => src2/renamed.md}\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"new_file.md", "src/renamed_py.py", "src2/renamed.md"}))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit, 10)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, model.View(), "new_file.md")

	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Type("r"))
	test.SimulateModel(model, test.Press(tea.KeyEnter))
}

func TestModel_Update_HandlesFilenamesWithBraces(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false\nM file{with}braces.txt\nA another{test}.go\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"file{with}braces.txt", "another{test}.go"}))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit, 10)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, model.View(), "file{with}braces.txt")

	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Type("r"))
	test.SimulateModel(model, test.Press(tea.KeyEnter))
}
