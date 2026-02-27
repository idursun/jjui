package revset

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/intents"
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

func TestModel_Update_IntentDoesNotAlterCurrentRevsetDisplay(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	ctx.CurrentRevset = "current"
	ctx.DefaultRevset = "default"
	model := New(ctx)
	test.SimulateModel(model, model.Init())
	test.SimulateModel(model, func() tea.Msg { return intents.CompletionMove{Delta: -1} })
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
