package revset

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestModel_Init(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	test.SimulateModel(model, model.Init())
}

func TestModel_Update_Up_SetsCurrentRevset(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.CurrentRevset = "current"
	ctx.DefaultRevset = "default"
	model := New(ctx)
	test.SimulateModel(model, model.Init())
	test.SimulateModel(model, test.Press(tea.KeyUp))
	assert.Contains(t, test.RenderImmediate(model, 80, 5), "current")
}

func TestModel_View_DisplaysCurrentRevset(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.CurrentRevset = "current"
	ctx.DefaultRevset = "default"
	model := New(ctx)
	assert.Contains(t, test.RenderImmediate(model, 80, 5), ctx.CurrentRevset)
}
