package details

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/stretchr/testify/assert"

	"github.com/idursun/jjui/test"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	Revision     = "ignored"
	StatusOutput = "false false $\nM file.txt\nA newfile.txt\n"
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

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file.txt")
}

func TestModel_Update_RestoresSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Restore(Revision, []string{"file.txt"}))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file.txt")

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

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file.txt")
	test.SimulateModel(model, test.Type("ri"))
}

func TestModel_Update_SplitsSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Split(Revision, []string{"file.txt"}, false))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file.txt")

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

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file.txt")

	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, func() tea.Msg {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s"), Alt: true}
	})
	test.SimulateModel(model, test.Press(tea.KeyEnter))
}

func TestModel_Update_HandlesMovedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false $\nR internal/ui/{revisions => }/file.go\nR {file => sub/newfile}\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"internal/ui/file.go", "sub/newfile"}))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file.go")

	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Type("r"))
	test.SimulateModel(model, test.Press(tea.KeyEnter))
}

func TestModel_Update_HandlesMovedFilesInDeepDirectories(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false false $\nR {src/new_file_3.md => new_file.md}\nR src/{new_file.py => renamed_py.py}\nR {src1/to_be_renamed.md => src2/renamed.md}\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"new_file.md", "src/renamed_py.py", "src2/renamed.md"}))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "new_file.md")

	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Type("r"))
	test.SimulateModel(model, test.Press(tea.KeyEnter))
}

func TestModel_Update_HandlesFilenamesWithBraces(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false $\nM file{with}braces.txt\nA another{test}.go\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"file{with}braces.txt", "another{test}.go"}))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file{with}braces.txt")

	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Press(tea.KeySpace))
	test.SimulateModel(model, test.Type("r"))
	test.SimulateModel(model, test.Press(tea.KeyEnter))
}

func TestModel_Refresh_IgnoreVirtuallySelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	test.SimulateModel(model, common.Refresh)
	for _, file := range model.files {
		assert.False(t, file.selected)
	}
}

func TestModel_Update_Quit(t *testing.T) {
	tests := []struct {
		name        string
		interaction tea.Cmd
		shouldQuit  bool
	}{
		{
			name:        "pressing q",
			interaction: test.Type("q"),
			shouldQuit:  true,
		},
		{
			name:        "pressing esc",
			interaction: test.Press(tea.KeyEscape),
			shouldQuit:  false,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			commandRunner := test.NewTestCommandRunner(t)

			model := NewOperation(test.NewTestContext(commandRunner), Commit)
			model.keymap.Quit = key.NewBinding(key.WithKeys("esc", "q"))
			var msgs []tea.Msg
			test.SimulateModel(model, testCase.interaction, func(msg tea.Msg) {
				msgs = append(msgs, msg)
			})

			if testCase.shouldQuit {
				assert.Contains(t, msgs, tea.QuitMsg{})
			} else {
				assert.NotContains(t, msgs, tea.QuitMsg{})
			}
		})
	}
}

func TestModel_createListItems(t *testing.T) {
	content := `false false false
false $
A test/file1
A test/file2
A test/file3
A test/file4`

	model := NewOperation(test.NewTestContext(test.NewTestCommandRunner(t)), Commit)
	files := model.createListItems(content, nil)
	assert.Len(t, files, 4)
}
