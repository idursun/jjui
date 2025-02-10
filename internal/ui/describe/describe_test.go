package describe

import (
	"bytes"
	"testing"
	"time"

	"github.com/idursun/jjui/test"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/stretchr/testify/assert"
)

func TestCancel(t *testing.T) {
	model := New(common.NewUICommands(&test.JJCommands{}), "revision", "description", 20)
	var cmd tea.Cmd
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.NotNil(t, cmd)
	msg := cmd()
	assert.Equal(t, common.CloseViewMsg{}, msg)
	assert.Equal(t, "revision", model.(Model).revision)
}

func TestEdit(t *testing.T) {
	commands := test.NewJJCommands()
	commands.ExpectSetDescription(t, "revision", "description changed")
	tm := teatest.NewTestModel(t, New(common.NewUICommands(commands), "revision", "description", 20))
	tm.Type(" changed")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("changed"))
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	commands.Verify(t)
}
