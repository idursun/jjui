package git

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_Push(t *testing.T) {
	c := test.NewTestContext(t)
	c.Expect(jj.GitPush())
	defer c.Verify()

	op := NewModel(c, nil, 0, 0)
	tm := teatest.NewTestModel(t, test.NewShell(op))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func Test_Fetch(t *testing.T) {
	c := test.NewTestContext(t)
	c.Expect(jj.GitFetch())
	defer c.Verify()

	op := NewModel(c, nil, 0, 0)
	tm := teatest.NewTestModel(t, test.NewShell(op))
	tm.Type("/")
	tm.Type("fetch")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func Test_loadBookmarks(t *testing.T) {
	const changeId = "changeid"
	c := test.NewTestContext(t)
	c.Expect(jj.BookmarkList(changeId)).SetOutput([]byte(`
feat/allow-new-bookmarks;.;false;false;false;83
feat/allow-new-bookmarks;origin;true;false;false;83
main;.;false;false;false;86
main;origin;true;false;false;86
test;.;false;false;false;d0
`))
	defer c.Verify()

	bookmarks := loadBookmarks(c, changeId)
	assert.Len(t, bookmarks, 3)
}
