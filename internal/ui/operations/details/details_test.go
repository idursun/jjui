package details

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	uiContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
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

type fakeChecks struct {
	checked map[string]bool
	toggles []string
}

func newFakeChecks() *fakeChecks { return &fakeChecks{checked: map[string]bool{}} }

func (f *fakeChecks) fileChecks() FileChecks {
	return FileChecks{IsChecked: f.isChecked, Toggle: f.toggle}
}

func (f *fakeChecks) isChecked(changeId string, file jj.FileName) bool {
	return f.checked[changeId+"/"+file.Path()]
}

func (f *fakeChecks) toggle(changeId string, file jj.FileName) {
	key := changeId + "/" + file.Path()
	f.checked[key] = !f.checked[key]
	f.toggles = append(f.toggles, key)
}

func loadOperation(t *testing.T, commandRunner *test.CommandRunner, output string) *Operation {
	t.Helper()
	return loadOperationWith(t, commandRunner, output, newFakeChecks())
}

func loadOperationWith(t *testing.T, commandRunner *test.CommandRunner, output string, checks *fakeChecks) *Operation {
	t.Helper()
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(revision)).SetOutput([]byte(output))

	operation := NewOperation(test.NewTestContext(commandRunner), commit, checks.fileChecks())
	test.SimulateModel(operation, operation.Init())
	operation.navigate(1, false)
	return operation
}

func TestOperation_OpensOnTheChangeIdAndMovesThroughFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(revision)).SetOutput([]byte(statusOutput))
	operation := NewOperation(test.NewTestContext(commandRunner), commit, newFakeChecks().fileChecks())
	test.SimulateModel(operation, operation.Init())

	assert.False(t, operation.FileFocused())
	assert.Nil(t, operation.Selection().Highlighted)

	operation.navigate(1, false)
	selected, ok := operation.Selection().Highlighted.(common.SelectedFile)
	require.True(t, ok)
	assert.Equal(t, "file.txt", selected.File.Path())

	operation.navigate(-1, false)
	assert.False(t, operation.FileFocused())
	assert.Nil(t, operation.Selection().Highlighted)
}

func TestOperation_DrawsTheCursorAtTheStartOfTheFocusedFile(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	operation := loadOperation(t, commandRunner, statusOutput)
	operation.navigate(1, false)
	box := layout.Box{R: layout.Rect(3, 5, 40, 2)}

	dl := render.NewDisplayContext()
	operation.RenderFileList(dl, box)
	require.NotNil(t, dl.Cursor())
	assert.Equal(t, 3, dl.Cursor().Position.X)
	assert.Equal(t, 6, dl.Cursor().Position.Y)

	operation.LeaveFiles()
	dl = render.NewDisplayContext()
	operation.RenderFileList(dl, box)
	assert.Nil(t, dl.Cursor())
}

func TestOperation_RefreshKeepsTheFocusOnTheChangeId(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	operation := loadOperation(t, commandRunner, statusOutput)
	operation.LeaveFiles()

	operation.setItems(operation.createListItems(statusOutput))

	assert.False(t, operation.FileFocused())
}

func TestOperation_FilterFocusesTheFirstMatch(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	operation := loadOperation(t, commandRunner, statusOutput)
	operation.LeaveFiles()

	operation.setFilter("new", true)

	require.True(t, operation.FileFocused())
	assert.Equal(t, "newfile.txt", operation.current().fileName.Path())
}

func TestOperation_InitLoadsStatus(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)

	operation := loadOperation(t, commandRunner, statusOutput)

	require.Len(t, operation.files, 2)
	assert.Equal(t, "file.txt", operation.files[0].fileName.Path())
	assert.Equal(t, "newfile.txt", operation.files[1].fileName.Path())
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
			commandRunner.Expect(jj.Restore(revision, []jj.FileName{jj.NewFileName("file.txt")}, tt.interactive))
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
			commandRunner.Expect(jj.Split(revision, []jj.FileName{jj.NewFileName("file.txt")}, tt.isParallel, false))
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

func TestOperation_InvertSelection(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	checks := newFakeChecks()
	operation := loadOperationWith(t, commandRunner, statusOutput, checks)

	_, handled := operation.HandleIntent(intents.DetailsInvertSelection{})
	require.True(t, handled)
	assert.False(t, checks.checked[revision+"/file.txt"])
	assert.True(t, checks.checked[revision+"/newfile.txt"])
	assert.Equal(t, []jj.FileName{jj.NewFileName("newfile.txt")}, operation.getSelectedFiles(true))

	_, handled = operation.HandleIntent(intents.DetailsInvertSelection{})
	require.True(t, handled)
	assert.True(t, checks.checked[revision+"/file.txt"])
	assert.False(t, checks.checked[revision+"/newfile.txt"])
	assert.Equal(t, []jj.FileName{jj.NewFileName("file.txt")}, operation.getSelectedFiles(true))
}

func TestOperation_InvertSelectionTreatsEmptySelectionAsVirtual(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	checks := newFakeChecks()
	operation := loadOperationWith(t, commandRunner, statusOutput, checks)
	checks.checked[revision+"/file.txt"] = true
	checks.checked[revision+"/newfile.txt"] = true

	_, handled := operation.HandleIntent(intents.DetailsInvertSelection{})
	require.True(t, handled)
	assert.Equal(t, []jj.FileName{jj.NewFileName("file.txt")}, operation.getSelectedFiles(true))
}

func TestOperation_InvertSelectionIncludesFilteredOutFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	checks := newFakeChecks()
	operation := loadOperationWith(t, commandRunner, statusOutput, checks)
	operation.setFilter("new", true)

	_, handled := operation.HandleIntent(intents.DetailsInvertSelection{})
	require.True(t, handled)
	assert.True(t, checks.checked[revision+"/file.txt"])
	assert.False(t, checks.checked[revision+"/newfile.txt"])
	assert.Equal(t, []jj.FileName{jj.NewFileName("file.txt")}, operation.getSelectedFiles(true))
}

func TestOperation_RefreshPreservesCheckedButNotHighlightedFile(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	checks := newFakeChecks()
	operation := loadOperationWith(t, commandRunner, statusOutput, checks)

	test.SimulateModel(operation, func() tea.Msg { return intents.DetailsToggleSelect{} })
	require.Equal(t, 1, operation.cursor)
	test.SimulateModel(operation, common.Refresh)

	assert.True(t, checks.checked[revision+"/file.txt"])
	assert.False(t, checks.checked[revision+"/newfile.txt"])
	assert.Equal(t, []jj.FileName{jj.NewFileName("file.txt")}, operation.getSelectedFiles(false))
	assert.Equal(t, []string{revision + "/file.txt"}, checks.toggles, "a reload must not write to the store")
}

func TestOperation_TogglesUseTheDisplayedChangeId(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	checks := newFakeChecks()
	operation := loadOperationWith(t, commandRunner, statusOutput, checks)

	commandRunner.Expect(jj.Status("other")).SetOutput([]byte(statusOutput))
	test.SimulateModel(operation, func() tea.Msg {
		return common.SelectionChangedMsg{Item: common.SelectedRevision{ChangeId: "other", CommitId: "other-commit"}}
	})
	operation.navigate(1, false)
	test.SimulateModel(operation, func() tea.Msg { return intents.DetailsToggleSelect{} })

	assert.Equal(t, []string{"other/newfile.txt"}, checks.toggles)
}

func TestOperation_OnlyTheDisplayedRevisionsChecksAreUsed(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	checks := newFakeChecks()
	checks.checked["other/file.txt"] = true
	checks.checked[revision+"/newfile.txt"] = true
	commandRunner.Expect(jj.Split(revision, []jj.FileName{jj.NewFileName("newfile.txt")}, false, false))
	commandRunner.Expect(jj.Restore(revision, []jj.FileName{jj.NewFileName("newfile.txt")}, false))
	operation := loadOperationWith(t, commandRunner, statusOutput, checks)

	require.Len(t, operation.Selection().Checked, 1)
	assert.Equal(t, "newfile.txt", operation.Selection().Checked[0].(uiContext.SelectedFile).File.Path())

	test.SimulateModel(operation, func() tea.Msg { return intents.DetailsSplit{} })
	test.SimulateModel(operation, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
	test.SimulateModel(operation, func() tea.Msg { return confirmation.CloseMsg{} })
	test.SimulateModel(operation, func() tea.Msg { return intents.DetailsRestore{} })
	test.SimulateModel(operation, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestOperation_HandleIntentUpdatesSelectionAfterNavigation(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	operation := loadOperation(t, commandRunner, statusOutput)

	selected, ok := operation.Selection().Highlighted.(common.SelectedFile)
	require.True(t, ok)
	assert.Equal(t, "file.txt", selected.File.Path())

	cmd, handled := operation.HandleIntent(intents.DetailsNavigate{Delta: 1})
	require.True(t, handled)
	test.SimulateModel(operation, cmd)

	selected, ok = operation.Selection().Highlighted.(common.SelectedFile)
	require.True(t, ok)
	assert.Equal(t, "newfile.txt", selected.File.Path())
}

func TestOperation_FilterLifecycle(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	operation := loadOperation(t, commandRunner, statusOutput)

	cmd, handled := operation.HandleIntent(intents.DetailsOpenFilter{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	assert.Equal(t, filterEditing, operation.filterState)
	assert.True(t, operation.IsEditing())

	operation.Update(tea.PasteMsg{Content: "new"})
	assert.Equal(t, 1, operation.VisibleLen())
	require.NotNil(t, operation.current())
	assert.Equal(t, "newfile.txt", operation.current().fileName.Path())

	_, handled = operation.HandleIntent(intents.DetailsApplyFilter{})
	require.True(t, handled)
	assert.Equal(t, filterApplied, operation.filterState)
	assert.False(t, operation.IsEditing())
	assert.Equal(t, 1, operation.VisibleLen())
	assert.Contains(t, test.Stripped(test.RenderImmediate(operation, 40, 2)), "/ new")

	_, handled = operation.HandleIntent(intents.DetailsCancelFilter{})
	require.True(t, handled)
	assert.Equal(t, filterOff, operation.filterState)
	assert.Equal(t, 2, operation.VisibleLen())
	assert.Empty(t, operation.filterInput.Value())
}

func TestOperation_FilterKeepsHiddenCheckedFilesInActions(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	checks := newFakeChecks()
	operation := loadOperationWith(t, commandRunner, statusOutput, checks)
	checks.checked[revision+"/file.txt"] = true
	operation.setFilter("new", true)

	assert.Equal(t, []jj.FileName{jj.NewFileName("file.txt")}, operation.getSelectedFiles(true))
	selection := operation.Selection()
	require.Len(t, selection.Checked, 1)
	checked, ok := selection.Checked[0].(uiContext.SelectedFile)
	require.True(t, ok)
	assert.Equal(t, "file.txt", checked.File.Path())
}

func TestOperation_FilterNoMatchesRendersInsideDetails(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	operation := loadOperation(t, commandRunner, statusOutput)
	operation.filterInput.SetValue("missing")
	operation.filterState = filterApplied
	operation.setFilter("missing", true)

	output := test.Stripped(test.RenderImmediate(operation, 40, 2))

	assert.Contains(t, output, "/ missing")
	assert.Contains(t, output, "No matching files")
	assert.Equal(t, 2, operation.EmbeddedHeight(commit, operations.RenderPositionAfter, 40))

	cmd, handled := operation.HandleIntent(intents.DetailsRestore{})
	assert.True(t, handled)
	assert.Nil(t, cmd)
	assert.Nil(t, operation.confirmation, "an empty filtered view must not run a whole-revision file operation")
}

func TestOperation_createListItems(t *testing.T) {
	operation := NewOperation(test.NewTestContext(test.NewTestCommandRunner(t)), commit, newFakeChecks().fileChecks())
	content := `true false true false true $
A added.txt
D deleted.txt
M modified.txt
R src/{old => renamed}.txt
C copied.txt`

	got := operation.createListItems(content)

	assert.Equal(t, []*item{
		{status: Added, name: "added.txt", fileName: jj.NewFileName("added.txt"), conflict: true},
		{status: Deleted, name: "deleted.txt", fileName: jj.NewFileName("deleted.txt")},
		{status: Modified, name: "modified.txt", fileName: jj.NewFileName("modified.txt"), conflict: true},
		{status: Renamed, name: "src/{ol => rename}d.txt", fileName: jj.NewFileName("src/renamed.txt")},
		{status: Copied, name: "copied.txt", fileName: jj.NewFileName("copied.txt"), conflict: true},
	}, got)
}

func TestOperationDisplaysWorkingDirectoryRelativeNamesButSelectsRepositoryPath(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	ctx.Location = "/work/repo"
	ctx.WorkingDirectory = "/work/repo/internal"
	operation := NewOperation(ctx, commit, newFakeChecks().fileChecks())

	items := operation.createListItems("false $\nM internal/ui/details.go\n")
	require.Len(t, items, 1)
	assert.Equal(t, "ui/details.go", items[0].name)
	assert.Equal(t, "internal/ui/details.go", items[0].fileName.Path())

	operation.setItems(items)
	operation.navigate(1, false)
	selected := operation.Selection().Highlighted.(common.SelectedFile)
	assert.Equal(t, "internal/ui/details.go", selected.File.Path())
}
