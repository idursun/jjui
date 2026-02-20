package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Theme_Simple(t *testing.T) {
	content := `
[ui]
theme = "my-theme"
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Equal(t, "my-theme", config.UI.Theme.Light)
	assert.Equal(t, "my-theme", config.UI.Theme.Dark)
}

func TestLoad_Theme_Nested(t *testing.T) {
	content := `
[ui.theme]
dark = "dark-theme"
light = "light-theme"
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Equal(t, "dark-theme", config.UI.Theme.Dark)
	assert.Equal(t, "light-theme", config.UI.Theme.Light)
}

func TestLoad_AutoRefreshInterval(t *testing.T) {
	content := `
[ui]
auto_refresh_interval = 5000
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Equal(t, 5000, config.UI.AutoRefreshInterval)
}

func TestLoad_FlashMessageDisplaySeconds(t *testing.T) {
	content := `
[ui]
flash_message_display_seconds = 10
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Equal(t, 10, config.UI.FlashMessageDisplaySeconds)
	assert.Equal(t, 10*time.Second, GetExpiringFlashMessageTimeout(config))
}

func TestLoad_Colors_StringAndObject(t *testing.T) {
	content := `
[ui.colors]
simple = "red"
complex = { fg = "blue", bg = "white", bold = true }
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)
	assert.Len(t, config.UI.Colors, 2)

	assert.Equal(t, "red", config.UI.Colors["simple"].Fg)
	assert.Equal(t, "", config.UI.Colors["simple"].Bg)
	assert.Nil(t, config.UI.Colors["simple"].Bold)

	assert.Equal(t, "blue", config.UI.Colors["complex"].Fg)
	assert.Equal(t, "white", config.UI.Colors["complex"].Bg)
	if assert.NotNil(t, config.UI.Colors["complex"].Bold) {
		assert.True(t, *config.UI.Colors["complex"].Bold)
	}
}

func TestLoad_Colors_ExplicitFalsePreserved(t *testing.T) {
	content := `
[ui.colors]
unset = { fg = "red" }
explicit_false = { fg = "blue", underline = false }
`
	config := &Config{}
	err := config.Load(content)
	assert.NoError(t, err)

	assert.Nil(t, config.UI.Colors["unset"].Underline)
	if assert.NotNil(t, config.UI.Colors["explicit_false"].Underline) {
		assert.False(t, *config.UI.Colors["explicit_false"].Underline)
	}
}
