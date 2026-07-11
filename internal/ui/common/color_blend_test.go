package common

import (
	"testing"

	"github.com/idursun/jjui/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlendHexColor_UsesGammaCorrectRGBBlend(t *testing.T) {
	got, ok, err := blendHexColor("#ff0000", "#000000", 0.5)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "#b40000", got)
}

func TestApplyThemeBackgroundBlend_BlendsSelectedBackgroundsTowardSurfaceBorderBackground(t *testing.T) {
	theme := map[string]config.Color{
		"selected":             {Fg: "#dcd7ba", Bg: "#363646"},
		"picker selected text": {Fg: "#dcd7ba", Bg: "#363646"},
		"unselected":           {Fg: "#dcd7ba", Bg: "#363646"},
		"ansi":                 {Fg: "#dcd7ba", Bg: "bright black"},
		"border":               {Bg: "#202020"},
		"picker border":        {Bg: "#202020"},
	}

	err := ApplyThemeBackgroundBlend(theme, 0.25, "", nil)
	require.NoError(t, err)

	assert.Equal(t, "#31313f", theme["selected"].Bg)
	assert.Equal(t, "#31313f", theme["picker selected text"].Bg)
	assert.Equal(t, "#363646", theme["unselected"].Bg)
	assert.Equal(t, "bright black", theme["ansi"].Bg)
}

func TestApplyThemeBackgroundBlend_UsesInheritedSurfaceBorderBackground(t *testing.T) {
	theme := map[string]config.Color{
		"git menu selected shortcut": {Fg: "#dcd7ba", Bg: "#363646"},
		"menu border":                {Bg: "#202020"},
	}

	err := ApplyThemeBackgroundBlend(theme, 0.25, "", nil)
	require.NoError(t, err)

	assert.Equal(t, "#31313f", theme["git menu selected shortcut"].Bg)
}

func TestApplyThemeBackgroundBlend_SkipsWhenSurfaceBorderHasNoBackground(t *testing.T) {
	theme := map[string]config.Color{
		"selected": {Fg: "#dcd7ba", Bg: "#363646"},
		"border":   {Fg: "#dcd7ba"},
	}

	err := ApplyThemeBackgroundBlend(theme, 0.25, "", nil)
	require.NoError(t, err)

	assert.Equal(t, "#363646", theme["selected"].Bg)
}

func TestApplyThemeBackgroundBlend_PrefersSurfaceBackgroundOverBorder(t *testing.T) {
	theme := map[string]config.Color{
		"picker":          {Bg: "#202020"},
		"picker border":   {Bg: "#ffffff"},
		"picker selected": {Bg: "#808080"},
	}

	err := ApplyThemeBackgroundBlend(theme, 0.25, "#000000", nil)
	require.NoError(t, err)

	assert.Equal(t, "#707070", theme["picker selected"].Bg)
}

func TestApplyThemeBackgroundBlend_UsesTerminalBackgroundForTransparentSurface(t *testing.T) {
	theme := map[string]config.Color{
		"picker selected": {Bg: "#808080"},
	}

	err := ApplyThemeBackgroundBlend(theme, 0.25, "#202020", nil)
	require.NoError(t, err)

	assert.Equal(t, "#707070", theme["picker selected"].Bg)
}

func TestApplyThemeBackgroundBlend_ResolvesDefaultSurfaceToTerminalBackground(t *testing.T) {
	theme := map[string]config.Color{
		"picker":          {Bg: "default"},
		"picker border":   {Bg: "#ffffff"},
		"picker selected": {Bg: "#808080"},
	}

	err := ApplyThemeBackgroundBlend(theme, 0.25, "#202020", nil)
	require.NoError(t, err)

	assert.Equal(t, "#707070", theme["picker selected"].Bg)
}

func TestApplyThemeBackgroundBlend_BlendsTerminalPaletteBackgroundsTowardSurfaceBorderBackground(t *testing.T) {
	theme := map[string]config.Color{
		"selected": {Fg: "#dcd7ba", Bg: "bright black"},
		"missing":  {Fg: "#dcd7ba", Bg: "bright red"},
		"border":   {Bg: "#202020"},
	}
	terminalPalette := map[int]string{
		8: "#808080",
	}

	err := ApplyThemeBackgroundBlend(theme, 0.25, "", terminalPalette)
	require.NoError(t, err)

	assert.Equal(t, "#707070", theme["selected"].Bg)
	assert.Equal(t, "bright red", theme["missing"].Bg)
}

func TestApplyThemeBackgroundBlend_ResolvesSurfaceBorderFromTerminalPalette(t *testing.T) {
	theme := map[string]config.Color{
		"selected": {Fg: "#dcd7ba", Bg: "#808080"},
		"border":   {Bg: "bright black"},
	}
	terminalPalette := map[int]string{
		8: "#202020",
	}

	err := ApplyThemeBackgroundBlend(theme, 0.25, "", terminalPalette)
	require.NoError(t, err)

	assert.Equal(t, "#707070", theme["selected"].Bg)
}
