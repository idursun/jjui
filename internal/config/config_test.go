package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoad_Colours(t *testing.T) {
	content := `
[ui.colors]
"text" = "white"
"selected" = { fg = "blue", bg = "black" }
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Len(t, config.UI.Colors, 2)
	assert.Equal(t, "white", config.UI.Colors["text"].Fg)
	assert.Equal(t, "blue", config.UI.Colors["selected"].Fg)
	assert.Equal(t, "black", config.UI.Colors["selected"].Bg)
}
