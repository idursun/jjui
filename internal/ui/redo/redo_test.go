package redo

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestConfirm(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLog(1))
	commandRunner.Expect(jj.Redo())
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetFrame(cellbuf.Rect(0, 0, 100, 20))
	model.Parent = common.NewViewNode(100, 20)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, model.View(), "redo")

	test.SimulateModel(model, test.Press(tea.KeyEnter))
}

func TestCancel(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLog(1))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetFrame(cellbuf.Rect(0, 0, 100, 20))
	model.Parent = common.NewViewNode(100, 20)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, model.View(), "redo")

	test.SimulateModel(model, test.Press(tea.KeyEsc))
}

func TestRedoNothingToRedo(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLog(1))
	commandRunner.Expect(jj.Redo()).SetError(errors.New("error: Nothing to redo"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetFrame(cellbuf.Rect(0, 0, 100, 20))
	model.Parent = common.NewViewNode(100, 20)
	test.SimulateModel(model, model.Init())

	var msgs []tea.Msg
	test.SimulateModel(model, test.Press(tea.KeyEnter), func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	var completed *common.CommandCompletedMsg
	for i := range msgs {
		if msg, ok := msgs[i].(common.CommandCompletedMsg); ok {
			completed = &msg
			break
		}
	}
	assert.NotNil(t, completed, "expected CommandCompletedMsg in %+v", msgs)
	if assert.NotNil(t, completed.Err) {
		assert.EqualError(t, completed.Err, "error: Nothing to redo")
	}
}
