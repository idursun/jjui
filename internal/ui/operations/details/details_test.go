package details

import (
	"bytes"
	"testing"

	"github.com/idursun/jjui/internal/jj"
	models2 "github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/idursun/jjui/test"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

const (
	Revision     = "ignored"
	StatusOutput = "false false\nM file.txt\nA newfile.txt\n"
)

var Commit = &models2.Commit{
	ChangeId: Revision,
	CommitId: Revision,
}

func TestModel_Init_ExecutesStatusCommand(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	defer commandRunner.Verify()

	appContext := context.NewAppContext(commandRunner, "")
	appContext.Revisions.Revisions.SetItems([]*models2.RevisionItem{
		{Row: models2.Row{Commit: Commit}},
	})
	appContext.Revisions.Revisions.Cursor = 0
	model := NewOperation(appContext, Commit)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
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

	appContext := context.NewAppContext(commandRunner, "")
	appContext.Revisions.Revisions.SetItems([]*models2.RevisionItem{
		{Row: models2.Row{Commit: Commit}},
	})
	appContext.Revisions.Revisions.Cursor = 0
	appContext.Files.SetItems([]*models2.RevisionFileItem{
		{
			Checkable: &models2.Checkable{Checked: false},
			Status:    0,
			Name:      "file.txt",
			FileName:  "file.txt",
			Conflict:  false,
		},
	})
	appContext.Files.Cursor = 0
	model := NewOperation(appContext, Commit)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.Id)
	tm := teatest.NewTestModel(t, model)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return model.confirmation == nil
	})
	tm.Quit()
}

func TestModel_Update_SplitsSelectedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte(StatusOutput))
	commandRunner.Expect(jj.Split(Revision, []string{"file.txt"}))
	defer commandRunner.Verify()

	appContext := context.NewAppContext(commandRunner, "")
	appContext.Revisions.Revisions.SetItems([]*models2.RevisionItem{
		{Row: models2.Row{Commit: Commit}},
	})
	appContext.Revisions.Revisions.Cursor = 0
	model := NewOperation(appContext, Commit)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.Id)
	tm := teatest.NewTestModel(t, model)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.txt"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return !viewManager.IsFocused(model.Id)
	})
	tm.Quit()
}

func TestModel_Update_HandlesMovedFiles(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(Revision)).SetOutput([]byte("false false\nR internal/ui/{revisions => }/file.go\nR {file => sub/newfile}\n"))
	commandRunner.Expect(jj.Restore(Revision, []string{"internal/ui/file.go", "sub/newfile"}))
	defer commandRunner.Verify()

	appContext := context.NewAppContext(commandRunner, "")
	appContext.Revisions.Revisions.SetItems([]*models2.RevisionItem{
		{Row: models2.Row{Commit: Commit}},
	})
	appContext.Revisions.Revisions.Cursor = 0
	model := NewOperation(appContext, Commit)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.Id)
	tm := teatest.NewTestModel(t, model)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("file.go"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return model.confirmation == nil
	})
	tm.Quit()
}
