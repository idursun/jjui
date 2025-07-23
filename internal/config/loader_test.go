package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTheme(t *testing.T) {
	themeData := []byte(`
title = { fg = "blue", bold = true }
selected = { fg = "white", bg = "blue" }
error = "red"
`)

	theme, err := loadTheme(themeData, nil)
	require.NoError(t, err)

	expected := map[string]Color{
		"title":    {Fg: "blue", Bold: true},
		"selected": {Fg: "white", Bg: "blue"},
		"error":    {Fg: "red"},
	}

	assert.Equal(t, expected, theme)
}

func TestLoadThemeWithBase(t *testing.T) {
	baseTheme := map[string]Color{
		"title":    {Fg: "green", Bold: true},
		"selected": {Fg: "cyan", Bg: "black"},
		"error":    {Fg: "red"},
		"border":   {Fg: "white"},
	}

	partialOverride := []byte(`
title = { fg = "magenta", bold = true }
selected = { fg = "yellow", bg = "blue" }
`)

	theme, err := loadTheme(partialOverride, baseTheme)
	require.NoError(t, err)

	expected := map[string]Color{
		"title":    {Fg: "magenta", Bold: true},
		"selected": {Fg: "yellow", Bg: "blue"},
		"error":    {Fg: "red"},
		"border":   {Fg: "white"},
	}

	assert.Equal(t, expected, theme)
}
