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
	statusOutput = "<<file status_char::M>><<file:: >><<file display_diff_path::file.txt>>\n<<file status_char::A>><<file:: >><<file display_diff_path::newfile.txt>>\n"
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

func TestOperation_ApplyCancelAndOptionSelectAreHandledOnlyWhenConsumed(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	t.Cleanup(commandRunner.Verify)
	operation := loadOperation(t, commandRunner, statusOutput)

	for _, intent := range []intents.Intent{intents.Apply{}, intents.Cancel{}, intents.OptionSelect{Delta: 1}} {
		_, handled := operation.HandleIntent(intent)
		assert.False(t, handled, "%T with no confirmation or filter", intent)
	}

	operation.HandleIntent(intents.DetailsOpenFilter{})
	operation.Update(tea.PasteMsg{Content: "new"})
	_, handled := operation.HandleIntent(intents.Apply{})
	assert.True(t, handled, "apply while editing the filter")
	assert.Equal(t, filterApplied, operation.filterState)
	_, handled = operation.HandleIntent(intents.Apply{})
	assert.False(t, handled, "apply with the filter already applied")
	_, handled = operation.HandleIntent(intents.Cancel{})
	assert.True(t, handled, "cancel clears the applied filter")
	assert.Equal(t, filterOff, operation.filterState)

	operation.HandleIntent(intents.DetailsRestore{})
	require.NotNil(t, operation.confirmation)
	for _, intent := range []intents.Intent{intents.OptionSelect{Delta: 1}, intents.Cancel{}} {
		_, handled = operation.HandleIntent(intent)
		assert.True(t, handled, "%T with a confirmation open", intent)
	}
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
	// Captured from jj 0.46 running jj.Status on a child of a 3-parent merge.
	content := "\x1b[38;5;2m<<log commit diff files map join file status_char::A>><<log commit diff files map join file:: >><<log commit diff files map join file display_diff_path::added.txt>><<log commit diff files map join::>>\n" +
		"<<log commit diff files map join file status_char::D>><<log commit diff files map join file:: >><<log commit diff files map join file display_diff_path::deleted.txt>><<log commit diff files map join::>>\n" +
		"<<log commit diff files map join file conflicted status_char::M>><<log commit diff files map join file conflicted:: >><<log commit diff files map join file conflicted display_diff_path::three.txt>><<log commit diff files map join file conflicted:: >><<log commit diff files map join file conflicted target conflict_side_count::3>><<log commit diff files map join::>>\n" +
		"<<log commit diff files map join file status_char::R>><<log commit diff files map join file:: >><<log commit diff files map join file display_diff_path::src/{old => renamed}.txt>><<log commit diff files map join::>>\n" +
		"<<log commit diff files map join file status_char::C>><<log commit diff files map join file:: >><<log commit diff files map join file display_diff_path::copied.txt>><<log commit::>>\n" +
		"<<log commit conflicted_files map join conflicted_file path::three.txt>><<log commit conflicted_files map join conflicted_file:: >><<log commit conflicted_files map join conflicted_file conflict_side_count::3>><<log commit conflicted_files map join::>>\n" +
		"<<log commit conflicted_files map join conflicted_file path::two.txt>><<log commit conflicted_files map join conflicted_file:: >><<log commit conflicted_files map join conflicted_file conflict_side_count::2>><<log commit::>>\x1b[39m\n"

	got := operation.createListItems(content)

	assert.Equal(t, []*item{
		{status: Added, name: "added.txt", fileName: jj.NewFileName("added.txt")},
		{status: Deleted, name: "deleted.txt", fileName: jj.NewFileName("deleted.txt")},
		{status: Modified, name: "three.txt", fileName: jj.NewFileName("three.txt"), conflict: true, conflictSides: 3},
		{status: Renamed, name: "src/{ol => rename}d.txt", fileName: jj.NewFileName("src/renamed.txt")},
		{status: Copied, name: "copied.txt", fileName: jj.NewFileName("copied.txt")},
		{status: Unchanged, name: "two.txt", fileName: jj.NewFileName("two.txt"), conflict: true, conflictSides: 2},
	}, got)
}

func TestOperation_createListItemsListsConflictsOfRevisionWithEmptyDiff(t *testing.T) {
	operation := NewOperation(test.NewTestContext(test.NewTestCommandRunner(t)), commit, newFakeChecks().fileChecks())
	content := "<<log commit::>>\n" +
		"<<log commit conflicted_files map join conflicted_file path::has space.txt>><<log commit conflicted_files map join conflicted_file:: >><<log commit conflicted_files map join conflicted_file conflict_side_count::2>><<log commit::>>\n"

	got := operation.createListItems(content)

	assert.Equal(t, []*item{
		{status: Unchanged, name: "has space.txt", fileName: jj.NewFileName("has space.txt"), conflict: true, conflictSides: 2},
	}, got)
	assert.Equal(t, "  has space.txt", got[0].Title())
}

func TestOperationDisplaysWorkingDirectoryRelativeNamesButSelectsRepositoryPath(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	ctx.Location = "/work/repo"
	ctx.WorkingDirectory = "/work/repo/internal"
	operation := NewOperation(ctx, commit, newFakeChecks().fileChecks())

	items := operation.createListItems("<<file status_char::M>><<file:: >><<file display_diff_path::internal/ui/details.go>>\n")
	require.Len(t, items, 1)
	assert.Equal(t, "ui/details.go", items[0].name)
	assert.Equal(t, "internal/ui/details.go", items[0].fileName.Path())

	operation.setItems(items)
	operation.navigate(1, false)
	selected := operation.Selection().Highlighted.(common.SelectedFile)
	assert.Equal(t, "internal/ui/details.go", selected.File.Path())
}
