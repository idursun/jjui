package git

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

var revision = models.RevisionItem{
	Checkable: nil,
	Row: models.Row{
		Commit: &models.Commit{CommitId: "revision"},
	},
	IsAffected: false,
}

func Test_Push(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitPushCommandArgs{}.GetArgs())
	defer commandRunner.Verify()

	ctx := context.NewAppContext(commandRunner, "")
	model := NewModel(ctx, nil)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.GetId())
	tm := teatest.NewTestModel(t, model)

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.VerifyCalled(jj.GitPushCommandArgs{}.GetArgs())
	})
	tm.Quit()
}

func Test_Fetch(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitFetchArgs{}.GetArgs())
	defer commandRunner.Verify()

	ctx := context.NewAppContext(commandRunner, "")
	model := NewModel(ctx, nil)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.GetId())
	tm := teatest.NewTestModel(t, model)

	tm.Type("/")
	tm.Type("fetch")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.VerifyCalled(jj.GitFetchArgs{}.GetArgs())
	})
	tm.Quit()
}

func Test_loadBookmarks(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListArgs{Revision: revision}.GetArgs()).SetOutput([]byte(`
feat/allow-new-bookmarks;.;false;false;false;83
feat/allow-new-bookmarks;origin;true;false;false;83
main;.;false;false;false;86
main;origin;true;false;false;86
test;.;false;false;false;d0
`))
	defer commandRunner.Verify()

	bookmarks := loadBookmarks(commandRunner, &revision)
	assert.Len(t, bookmarks, 3)
}

func Test_PushChange(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	// Expect bookmark list to be loaded since we have a changeId
	commandRunner.Expect(jj.BookmarkListArgs{Revision: revision}.GetArgs()).SetOutput([]byte(""))
	gitPushArgs := jj.GitPushCommandArgs{Change: revision}
	commandRunner.Expect(gitPushArgs.GetArgs())
	defer commandRunner.Verify()

	ctx := context.NewAppContext(commandRunner, "")
	model := NewModel(ctx, &revision)
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.GetId())
	tm := teatest.NewTestModel(t, model)

	// Filter for the exact item and ensure selection is at index 0
	tm.Type("/")
	tm.Type("git push --change")
	tm.Send(tea.KeyMsg{Type: tea.KeyDown}) // Ensure first item is selected
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.VerifyCalled(gitPushArgs.GetArgs())
	})
	tm.Quit()
}
