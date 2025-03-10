package undo

import (
	"bytes"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
	"testing"
	"time"
)

func TestConfirm(t *testing.T) {
	c := test.NewTestContext(t)
	c.Expect(jj.Undo())
	defer c.Verify()

	operation, _ := NewOperation(c)
	tm := teatest.NewTestModel(t, test.OperationHost{Operation: operation})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("undo"))
	})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestCancel(t *testing.T) {
	c := test.NewTestContext(t)
	defer c.Verify()

	operation, _ := NewOperation(c)
	model := test.OperationHost{Operation: operation}

	tm := teatest.NewTestModel(t, model)
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
