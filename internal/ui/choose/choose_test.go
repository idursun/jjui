package choose

import (
	"testing"

	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWithTitle(t *testing.T) {
	options := []string{"Option 1", "Option 2", "Option 3"}
	title := "Choose an option"
	model := NewWithTitle(options, title)

	assert.NotEmpty(t, model.title)
}

func TestModel_View(t *testing.T) {
	options := []string{"Option 1", "Option 2", "Option 3"}
	title := "Choose an option"
	model := NewWithTitle(options, title)
	test.SimulateModel(model, model.Init())
	output := test.RenderImmediate(model, 80, 20)
	require.NotEmpty(t, output)

	assert.Contains(t, output, title)
	for _, option := range options {
		assert.Contains(t, output, option)
	}
}
