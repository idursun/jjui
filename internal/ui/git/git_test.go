package git

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func Test_Push(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitPush())
	defer commandRunner.Verify()

	op := NewModel(test.NewTestContext(commandRunner), jj.NewSelectedRevisions(), 0, 0)
	tm := teatest.NewTestModel(t, op)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.IsVerified()
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func Test_Fetch(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitFetch())
	defer commandRunner.Verify()

	op := NewModel(test.NewTestContext(commandRunner), jj.NewSelectedRevisions(), 0, 0)
	tm := teatest.NewTestModel(t, op)
	tm.Type("/")
	tm.Type("fetch")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.IsVerified()
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func Test_loadBookmarks(t *testing.T) {
	const changeId = "changeid"
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkList(changeId)).SetOutput([]byte(`
feat/allow-new-bookmarks;.;false;false;false;83
feat/allow-new-bookmarks;origin;true;false;false;83
main;.;false;false;false;86
main;origin;true;false;false;86
test;.;false;false;false;d0
`))
	defer commandRunner.Verify()

	bookmarks := loadBookmarks(commandRunner, changeId)
	assert.Len(t, bookmarks, 3)
}

func Test_PushChange(t *testing.T) {
	const changeId = "abc123"
	commandRunner := test.NewTestCommandRunner(t)
	// Expect bookmark list to be loaded since we have a changeId
	commandRunner.Expect(jj.BookmarkList(changeId)).SetOutput([]byte(""))
	commandRunner.Expect(jj.GitPush("--change", changeId))
	defer commandRunner.Verify()

	op := NewModel(test.NewTestContext(commandRunner), jj.NewSelectedRevisions(&jj.Commit{ChangeId: changeId}), 0, 0)
	tm := teatest.NewTestModel(t, op)
	// Filter for the exact item and ensure selection is at index 0
	tm.Type("/")
	tm.Type("git push --change")
	tm.Send(tea.KeyMsg{Type: tea.KeyDown}) // Ensure first item is selected
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.IsVerified()
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
