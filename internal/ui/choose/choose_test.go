package choose

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWithTitle(t *testing.T) {
	options := []string{"Option 1", "Option 2", "Option 3"}
	title := "Choose an option"
	model := NewWithTitle(options, title, false)

	assert.NotEmpty(t, model.title)
}

func TestModel_View(t *testing.T) {
	options := []string{"Option 1", "Option 2", "Option 3"}
	title := "Choose an option"
	model := NewWithTitle(options, title, false)
	test.SimulateModel(model, model.Init())
	output := test.RenderImmediate(model, 80, 20)
	require.NotEmpty(t, output)

	assert.Contains(t, output, title)
	for _, option := range options {
		assert.Contains(t, output, option)
	}
}

func TestModel_Filter(t *testing.T) {
	options := []string{"foo", "bar", "baz"}
	model := NewWithTitle(options, "Filter Test", true)

	// Simulate typing '/'
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, model.filtering)

	// Simulate typing 'b' — Update calls filterOptions internally
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	assert.Contains(t, model.filteredOptions, "bar")
	assert.Contains(t, model.filteredOptions, "baz")
	assert.NotContains(t, model.filteredOptions, "foo")
}

func TestModel_Ordered_View(t *testing.T) {
	options := []string{"alpha", "beta", "gamma"}
	model := NewWithOptions(options, "Ordered Test", false, true)
	test.SimulateModel(model, model.Init())
	output := test.RenderImmediate(model, 80, 20)

	assert.Contains(t, output, "1. alpha")
	assert.Contains(t, output, "2. beta")
	assert.Contains(t, output, "3. gamma")
}

func TestModel_Ordered_DigitSelect(t *testing.T) {
	options := []string{"alpha", "beta", "gamma"}
	model := NewWithOptions(options, "Ordered Test", false, true)

	cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	require.NotNil(t, cmd)

	msg := cmd()
	selected, ok := msg.(SelectedMsg)
	require.True(t, ok)
	assert.Equal(t, "beta", selected.Value)
}

func TestModel_Ordered_DigitOutOfRange(t *testing.T) {
	options := []string{"alpha", "beta"}
	model := NewWithOptions(options, "Ordered Test", false, true)

	cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	assert.Nil(t, cmd)
}

func TestModel_NonOrdered_DigitIgnored(t *testing.T) {
	options := []string{"alpha", "beta", "gamma"}
	model := NewWithOptions(options, "Test", false, false)

	cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	assert.Nil(t, cmd)
}
