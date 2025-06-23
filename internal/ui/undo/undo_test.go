package undo

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
)

func TestConfirm(t *testing.T) {
	c := test.NewTestContext(t)
	c.Expect(jj.OpLog(1))
	c.Expect(jj.Undo())
	defer c.Verify()

	model := NewModel(c)
	tm := teatest.NewTestModel(t, test.NewShell(model))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("undo"))
	})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestCancel(t *testing.T) {
	c := test.NewTestContext(t)
	c.Expect(jj.OpLog(1))
	defer c.Verify()

	tm := teatest.NewTestModel(t, test.NewShell(NewModel(c)))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("undo"))
	})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("closed"))
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
