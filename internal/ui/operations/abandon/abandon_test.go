package abandon

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
	"github.com/idursun/jjui/test"
)

var revision = models.RevisionItem{
	Checkable: nil,
	Row: models.Row{
		Commit: &models.Commit{ChangeId: "a"},
	},
	IsAffected: false,
}

var revisions = jj.NewSelectedRevisions(&revision)

func Test_Accept(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.AbandonArgs{Revisions: revisions, RetainBookmarks: true}.GetArgs())
	defer commandRunner.Verify()

	appContext := context.NewAppContext(commandRunner, "")
	model := NewOperation(appContext, revisions)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.Id)
	tm := teatest.NewTestModel(t, model)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("abandon"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return !viewManager.IsFocused(model.Id)
	})
	tm.Quit()
}

func Test_Cancel(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	appContext := context.NewAppContext(commandRunner, "")
	model := NewOperation(appContext, revisions)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.Id)
	tm := teatest.NewTestModel(t, model)
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return !viewManager.IsFocused(model.Id)
	})
}
