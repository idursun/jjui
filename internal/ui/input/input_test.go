package input

import (
	"testing"

	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWithTitle(t *testing.T) {
	title := "Enter text"
	prompt := "Text: "
	model := NewWithTitle(title, prompt)

	assert.NotEmpty(t, model.title)
	assert.NotEmpty(t, model.prompt)
	assert.Equal(t, title, model.title)
	assert.Equal(t, prompt, model.prompt)
}

func TestNew(t *testing.T) {
	model := New()

	assert.Empty(t, model.title)
	assert.Empty(t, model.prompt)
	assert.Equal(t, "> ", model.input.Prompt)
}

func TestModel_View(t *testing.T) {
	title := "Enter text"
	prompt := "Text: "
	model := NewWithTitle(title, prompt)
	test.SimulateModel(model, model.Init())
	output := test.RenderImmediate(model, 80, 20)
	require.NotEmpty(t, output)

	assert.Contains(t, output, title)
	assert.Contains(t, output, prompt)
}

func TestModel_View_NoTitle(t *testing.T) {
	prompt := "Text: "
	model := NewWithTitle("", prompt)
	test.SimulateModel(model, model.Init())
	output := test.RenderImmediate(model, 80, 20)
	require.NotEmpty(t, output)

	assert.Contains(t, output, prompt)
}

func TestModel_View_OnlyInput(t *testing.T) {
	model := New()
	test.SimulateModel(model, model.Init())
	output := test.RenderImmediate(model, 80, 20)
	require.NotEmpty(t, output)

	assert.Contains(t, output, "> ")
}

func TestModel_SelectCurrent_WithText(t *testing.T) {
	model := New()
	test.SimulateModel(model, model.Init())
	test.SimulateModel(model, test.Type("test message"))
	cmd := model.selectCurrent()
	require.NotNil(t, cmd)

	msg := cmd()
	selectedMsg, ok := msg.(SelectedMsg)
	require.True(t, ok)
	assert.Equal(t, "test message", selectedMsg.Value)
}

func TestModel_SelectCurrent_Empty(t *testing.T) {
	model := New()
	cmd := model.selectCurrent()
	require.NotNil(t, cmd)

	msg := cmd()
	selectedMsg, ok := msg.(SelectedMsg)
	require.True(t, ok)
	assert.Equal(t, "", selectedMsg.Value)
}
