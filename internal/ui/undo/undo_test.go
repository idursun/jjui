package undo

import (
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
	commandRunner.Expect(jj.Undo())
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetFrame(cellbuf.Rect(0, 0, 100, 20))
	model.Parent = common.NewViewNode(100, 20)
	test.SimulateModel(model, model.Init())
	assert.Contains(t, model.View(), "undo")

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
	assert.Contains(t, model.View(), "undo")

	test.SimulateModel(model, test.Press(tea.KeyEsc))
}
