package bookmark

import (
	"bytes"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"jjui/internal/ui/common"
	"jjui/test"
	"testing"
	"time"
)

func TestSetBookmarkModel_Update(t *testing.T) {
	commands := test.NewJJCommands()
	commands.ExpectSetBookmark(t, "revision", "name")
	tm := teatest.NewTestModel(t, NewSetBookmark(common.NewUICommands(commands), "revision"))
	tm.Type("name")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("name"))
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	commands.Verify(t)
}
