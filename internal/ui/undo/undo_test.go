package undo

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestConfirm(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLog(1))
	commandRunner.Expect(jj.Undo())
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "undo")

	test.SimulateModel(model, func() tea.Msg { return intents.Apply{} })
}

func TestCancel(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLog(1))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.Init())
	assert.Contains(t, test.RenderImmediate(model, 100, 20), "undo")

	test.SimulateModel(model, func() tea.Msg { return intents.Cancel{} })
}

func TestUndo_ZIndex_RendersAbovePreview(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.OpLog(1))

	model := NewModel(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.Init())

	dl := render.NewDisplayContext()
	box := layout.Box{R: cellbuf.Rect(0, 0, 100, 40)}
	model.ViewRect(dl, box)

	draws := dl.DrawList()
	assert.NotEmpty(t, draws, "Expected undo confirmation to produce draw operations")

	for _, draw := range draws {
		assert.GreaterOrEqual(t, draw.Z, render.ZDialogs,
			"Undo confirmation must render above preview panel (ZDialogs)")
	}
}
