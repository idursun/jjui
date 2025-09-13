package bookmark

import (
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"

	"github.com/idursun/jjui/test"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

var revision = models.RevisionItem{
	Checkable: nil,
	Row: models.Row{
		Commit: &models.Commit{CommitId: "revision"},
	},
	IsAffected: false,
}

func TestSetBookmarkModel_Update(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovableArgs{Revision: revision}.GetArgs())
	bookmarkSetArgs := jj.BookmarkSetArgs{Revision: revision, Bookmark: "name"}
	commandRunner.Expect(bookmarkSetArgs.GetArgs())
	defer commandRunner.Verify()

	appContext := context.NewAppContext(commandRunner, "")
	model := NewSetBookmarkOperation(appContext, &revision)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.GetId())
	tm := teatest.NewTestModel(t, model)
	tm.Type("name")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.VerifyCalled(bookmarkSetArgs.GetArgs())
	})
}
