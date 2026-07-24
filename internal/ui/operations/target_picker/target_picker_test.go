package target_picker

import (
	"testing"

	"github.com/idursun/jjui/internal/jj/source"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticSource struct {
	items []source.Item
}

func (s staticSource) Fetch(_ source.Runner) ([]source.Item, error) {
	return s.items, nil
}

func TestAcceptReturnsItemValue(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	defer runner.Verify()
	model := NewModel(
		test.NewTestContext(runner),
		"payload",
		staticSource{items: []source.Item{
			{Name: "display label", Value: "selection-value", Kind: source.KindRevision},
		}},
	)

	load := model.Init()
	require.NotNil(t, load)
	require.NotNil(t, model.Update(load()))

	accept := model.accept(false)
	require.NotNil(t, accept)
	message, ok := accept().(TargetSelectedMsg)
	require.True(t, ok)
	assert.Equal(t, "selection-value", message.Target)
	assert.Equal(t, "payload", message.Payload)
}

func TestAcceptFallsBackToItemName(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	defer runner.Verify()
	model := NewModel(
		test.NewTestContext(runner),
		nil,
		staticSource{items: []source.Item{{Name: "display-and-selection"}}},
	)

	require.NotNil(t, model.Update(model.Init()()))
	message, ok := model.accept(false)().(TargetSelectedMsg)
	require.True(t, ok)
	assert.Equal(t, "display-and-selection", message.Target)
}
