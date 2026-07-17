package details

import (
	"testing"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"

	"github.com/idursun/jjui/test"

	tea "charm.land/bubbletea/v2"
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
	commandRunner.Expect(jj.Restore(Revision, []string{"file.txt"}, false))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file.txt")

	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	test.SimulateModel(model, func() tea.Msg { return intents.DetailsRestore{} })
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestModel_Update_RestoresInteractively(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Restore(Revision, []string{"file.txt"}, true))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file.txt")
	test.SimulateModel(model, func() tea.Msg { return intents.DetailsRestore{} })
	test.SimulateModel(model, func() tea.Msg {
		return confirmation.SelectOptionMsg{Index: 1}
	})
}

func TestModel_Update_SplitsSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Split(Revision, []string{"file.txt"}, false, false))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file.txt")

	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	test.SimulateModel(model, func() tea.Msg { return intents.DetailsSplit{} })
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestModel_Update_SplitHintsFollowCheckedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())

	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	test.SimulateModel(model, func() tea.Msg { return intents.DetailsSplit{} })

	rendered := test.RenderImmediate(model, 100, 20)
	assert.Contains(t, rendered, "file.txt stays as is")
	assert.Contains(t, rendered, "newfile.txt moves to the new revision")
}

func TestModel_Update_ParallelSplitsSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Split(Revision, []string{"file.txt"}, true, false))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file.txt")

	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	test.SimulateModel(model, func() tea.Msg { return intents.DetailsSplit{IsParallel: true} })
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestModel_Update_HandlesMovedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false $\nR internal/ui/{revisions => }/file.go\nR {file => sub/newfile}\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"internal/ui/file.go", "sub/newfile"}, false))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file.go")

	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	test.SimulateModel(model, func() tea.Msg { return intents.DetailsRestore{} })
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestModel_Update_HandlesMovedFilesInDeepDirectories(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false false $\nR {src/new_file_3.md => new_file.md}\nR src/{new_file.py => renamed_py.py}\nR {src1/to_be_renamed.md => src2/renamed.md}\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"new_file.md", "src/renamed_py.py", "src2/renamed.md"}, false))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "new_file.md")

	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	test.SimulateModel(model, func() tea.Msg { return intents.DetailsRestore{} })
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestModel_Update_HandlesFilenamesWithBraces(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false $\nM file{with}braces.txt\nA another{test}.go\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"file{with}braces.txt", "another{test}.go"}, false))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "file{with}braces.txt")

	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	test.SimulateModel(model, func() tea.Msg { return intents.DetailsToggleSelect{} })
	test.SimulateModel(model, func() tea.Msg { return intents.DetailsRestore{} })
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
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

func TestModel_HandleIntent_UpdatesSelectedFileWhenCursorMoves(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	test.SimulateModel(model, model.Init())

	selected, ok := model.Selection().Highlighted.(common.SelectedFile)
	assert.True(t, ok)
	assert.Equal(t, "file.txt", selected.File)

	cmd, handled := model.HandleIntent(intents.DetailsNavigate{Delta: 1})
	assert.True(t, handled)
	test.SimulateModel(model, cmd)

	selected, ok = model.Selection().Highlighted.(common.SelectedFile)
	assert.True(t, ok)
	assert.Equal(t, "newfile.txt", selected.File)
}

func TestDetailsList_SelectedRowsUseStatusSpecificSelectedStyles(t *testing.T) {
	originalPalette := common.DefaultPalette
	palette := common.NewPalette()
	palette.Update(map[string]config.Color{
		"revisions details text":            {Fg: "#ffffff", Bg: "#000000"},
		"revisions details:selected":        {Bg: "#220044", Bold: boolPtr(true)},
		"revisions details added:selected":  {Fg: "#55ff99", Bg: "#220044", Bold: boolPtr(true)},
		"revisions details dimmed:selected": {Fg: "#ccccff", Bg: "#220044"},
	})
	common.DefaultPalette = palette
	defer func() { common.DefaultPalette = originalPalette }()

	list := NewDetailsList()
	list.files = []*item{{status: Added, name: "new.txt", fileName: "new.txt"}}
	list.cursor = 0

	dl := render.NewDisplayContext()
	list.RenderFileList(dl, layout.NewBox(layout.Rect(0, 0, 20, 1)))

	buf := uv.NewScreenBuffer(20, 1)
	dl.Render(buf)

	cell := buf.CellAt(0, 0)
	wantFg, wantBg := renderExpectedCellColors(t,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#55ff99")).Background(lipgloss.Color("#220044")).Render("A"))
	assert.Equal(t, wantFg, cell.Style.Fg)
	assert.Equal(t, wantBg, cell.Style.Bg)
}

func TestModel_Update_Quit(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	model := NewOperation(test.NewTestContext(commandRunner), Commit)
	var msgs []tea.Msg
	test.SimulateModel(model, func() tea.Msg { return intents.Quit{} }, func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	assert.Contains(t, msgs, tea.QuitMsg{})
}

func boolPtr(v bool) *bool { return &v }

func renderExpectedCellColors(t *testing.T, content string) (any, any) {
	t.Helper()
	dl := render.NewDisplayContext()
	dl.AddDraw(layout.Rect(0, 0, 1, 1), content, 0)
	buf := uv.NewScreenBuffer(1, 1)
	dl.Render(buf)
	cell := buf.CellAt(0, 0)
	return cell.Style.Fg, cell.Style.Bg
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
