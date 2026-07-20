package details

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	revision     = "ignored"
	statusOutput = "false false $\nM file.txt\nA newfile.txt\n"
)

var commit = &jj.Commit{
	ChangeId: revision,
	CommitId: revision,
}

func loadOperation(t *testing.T, commandRunner *test.CommandRunner, output string) *Operation {
	t.Helper()
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(revision)).SetOutput([]byte(output))

	operation := NewOperation(test.NewTestContext(commandRunner), commit)
	test.SimulateModel(operation, operation.Init())
	return operation
}

func TestOperation_InitLoadsStatus(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)

	operation := loadOperation(t, commandRunner, statusOutput)

	require.Len(t, operation.files, 2)
	assert.Equal(t, "file.txt", operation.files[0].fileName)
	assert.Equal(t, "newfile.txt", operation.files[1].fileName)
}

func TestOperation_Restore(t *testing.T) {
	for _, tt := range []struct {
		name        string
		interactive bool
		option      int
		checkFile   bool
	}{
		{name: "checked file", option: 0, checkFile: true},
		{name: "highlighted file interactively", interactive: true, option: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			commandRunner := test.NewTestCommandRunner(t)
			t.Cleanup(commandRunner.Verify)
			commandRunner.Expect(jj.Restore(revision, []string{"file.txt"}, tt.interactive))
			operation := loadOperation(t, commandRunner, statusOutput)

			if tt.checkFile {
				test.SimulateModel(operation, func() tea.Msg { return intents.DetailsToggleSelect{} })
			}
			test.SimulateModel(operation, func() tea.Msg { return intents.DetailsRestore{} })
			test.SimulateModel(operation, func() tea.Msg {
				return confirmation.SelectOptionMsg{Index: tt.option}
			})
		})
	}
}

func TestOperation_Split(t *testing.T) {
	for _, tt := range []struct {
		name       string
		isParallel bool
	}{
		{name: "normal"},
		{name: "parallel", isParallel: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			commandRunner := test.NewTestCommandRunner(t)
			t.Cleanup(commandRunner.Verify)
			commandRunner.Expect(jj.Split(revision, []string{"file.txt"}, tt.isParallel, false))
			operation := loadOperation(t, commandRunner, statusOutput)

			test.SimulateModel(operation, func() tea.Msg { return intents.DetailsToggleSelect{} })
			test.SimulateModel(operation, func() tea.Msg {
				return intents.DetailsSplit{IsParallel: tt.isParallel}
			})
			assert.Equal(t, "stays as is", operation.selectedHint)
			assert.Equal(t, "moves to the new revision", operation.unselectedHint)
			test.SimulateModel(operation, func() tea.Msg {
				return confirmation.SelectOptionMsg{Index: 0}
			})
		})
	}
}

func TestOperation_RefreshPreservesCheckedButNotHighlightedFile(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	operation := loadOperation(t, commandRunner, statusOutput)

	test.SimulateModel(operation, func() tea.Msg { return intents.DetailsToggleSelect{} })
	require.Equal(t, 1, operation.cursor)
	test.SimulateModel(operation, common.Refresh)

	assert.True(t, operation.files[0].selected)
	assert.False(t, operation.files[1].selected)
}

func TestOperation_HandleIntentUpdatesSelectionAfterNavigation(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	operation := loadOperation(t, commandRunner, statusOutput)

	selected, ok := operation.Selection().Highlighted.(common.SelectedFile)
	require.True(t, ok)
	assert.Equal(t, "file.txt", selected.File)

	cmd, handled := operation.HandleIntent(intents.DetailsNavigate{Delta: 1})
	require.True(t, handled)
	test.SimulateModel(operation, cmd)

	selected, ok = operation.Selection().Highlighted.(common.SelectedFile)
	require.True(t, ok)
	assert.Equal(t, "newfile.txt", selected.File)
}

func TestOperation_createListItems(t *testing.T) {
	operation := NewOperation(test.NewTestContext(test.NewTestCommandRunner(t)), commit)
	content := `true false true false true $
A added.txt
D deleted.txt
M modified.txt
R src/{old => renamed}.txt
C copied.txt`

	got := operation.createListItems(content, []string{"deleted.txt"})

	assert.Equal(t, []*item{
		{status: Added, name: "added.txt", fileName: "added.txt", conflict: true},
		{status: Deleted, name: "deleted.txt", fileName: "deleted.txt", selected: true},
		{status: Modified, name: "modified.txt", fileName: "modified.txt", conflict: true},
		{status: Renamed, name: "src/{old => renamed}.txt", fileName: "src/renamed.txt"},
		{status: Copied, name: "copied.txt", fileName: "copied.txt", conflict: true},
	}, got)
}
