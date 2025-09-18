package undo

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/view"
	"github.com/idursun/jjui/test"
)

func TestConfirm(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLogArgs{
		NoGraph:         false,
		Limit:           1,
		GlobalArguments: jj.GlobalArguments{IgnoreWorkingCopy: true, Color: "always"},
	}.GetArgs())
	commandRunner.Expect(jj.UndoArgs{}.GetArgs())
	defer commandRunner.Verify()

	model := NewModel(context.NewAppContext(commandRunner, ""))
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.GetId())
	tm := teatest.NewTestModel(t, model)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("undo"))
	})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return !viewManager.IsFocused(model.GetId())
	})
	tm.Quit()
}

func TestCancel(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLogArgs{
		NoGraph:         false,
		Limit:           1,
		GlobalArguments: jj.GlobalArguments{IgnoreWorkingCopy: true, Color: "always"},
	}.GetArgs())
	defer commandRunner.Verify()

	model := NewModel(context.NewAppContext(commandRunner, ""))
	viewManager := view.NewViewManager()
	_ = viewManager.CreateView(model)
	viewManager.FocusView(model.GetId())
	tm := teatest.NewTestModel(t, model)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("undo"))
	})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return !viewManager.IsFocused(model.GetId())
	})
	tm.Quit()
}
