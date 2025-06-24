package bookmark

import (
	"github.com/idursun/jjui/internal/jj"
	"testing"
	"time"

	"github.com/idursun/jjui/test"

	tea "github.com/charmbracelet/bubbletea/v2"
)

func TestSetBookmarkModel_Update(t *testing.T) {
	c := test.NewTestContext(t)
	c.Expect(jj.BookmarkSet("revision", "name"))
	defer c.Verify()

	op, _ := NewSetBookmarkOperation(c, "revision")
	host := test.OperationHost{Operation: op}
	tm := teatest.NewTestModel(t, host)
	tm.Type("name")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
