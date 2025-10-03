package redo

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/test"
)

func TestConfirm(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLog(1))
	commandRunner.Expect(jj.Redo())
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	tm := teatest.NewTestModel(t, model)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("redo"))
	})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.IsVerified()
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestCancel(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLog(1))
	defer commandRunner.Verify()

	tm := teatest.NewTestModel(t, NewModel(test.NewTestContext(commandRunner)))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("redo"))
	})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return commandRunner.IsVerified()
	})
	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestRedoNothingToRedo(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLog(1))
	commandRunner.Expect(jj.Redo()).SetError(errors.New("Error: Nothing to redo."))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	msgs := collectMsgs(cmd)
	if len(msgs) == 0 {
		t.Fatalf("expected command messages, got none")
	}

	var completed *common.CommandCompletedMsg
	for i := range msgs {
		if msg, ok := msgs[i].(common.CommandCompletedMsg); ok {
			completed = &msg
			break
		}
	}

	if completed == nil {
		t.Fatalf("expected CommandCompletedMsg in %+v", msgs)
	}

	if completed.Err == nil {
		t.Fatalf("expected error message, got nil")
	}

	if completed.Err.Error() != "Error: Nothing to redo." {
		t.Fatalf("unexpected error message: %v", completed.Err)
	}
}

func collectMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}

	switch m := msg.(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range m {
			out = append(out, collectMsgs(c)...)
		}
		return out
	}

	val := reflect.ValueOf(msg)
	if val.Kind() == reflect.Slice && val.Type().Elem() == reflect.TypeOf((tea.Cmd)(nil)) {
		var out []tea.Msg
		for i := 0; i < val.Len(); i++ {
			out = append(out, collectMsgs(val.Index(i).Interface().(tea.Cmd))...)
		}
		return out
	}

	return []tea.Msg{msg}
}
