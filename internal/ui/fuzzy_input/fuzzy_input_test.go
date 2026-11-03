package fuzzy_input

import (
	"testing"

	"charm.land/bubbles/v2/textinput"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/stretchr/testify/assert"
)

func TestSuggestCycleIntent(t *testing.T) {
	orig := config.Current.Suggest.Exec.Mode
	defer func() { config.Current.Suggest.Exec.Mode = orig }()
	config.Current.Suggest.Exec.Mode = "off"

	input := textinput.New()
	model := New(&input, []string{"abc"})

	_ = model.Update(intents.SuggestCycle{})
	assert.Equal(t, config.SuggestModeFuzzy, model.suggestMode)
	_ = model.Update(intents.SuggestCycle{})
	assert.Equal(t, config.SuggestModeRegex, model.suggestMode)
	_ = model.Update(intents.SuggestCycle{})
	assert.Equal(t, config.SuggestModeOff, model.suggestMode)
}

func TestSuggestNavigateIntent(t *testing.T) {
	orig := config.Current.Suggest.Exec.Mode
	defer func() { config.Current.Suggest.Exec.Mode = orig }()
	config.Current.Suggest.Exec.Mode = "off"

	input := textinput.New()
	model := New(&input, []string{"fetch", "push"})

	assert.Equal(t, 0, model.cursor)

	_ = model.Update(intents.SuggestNavigate{Delta: 1})
	assert.Equal(t, 1, model.cursor)
}
