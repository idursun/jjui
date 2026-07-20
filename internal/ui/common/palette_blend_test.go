package common

import (
	"testing"

	"github.com/idursun/jjui/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyThemeBackgroundBlend_BlendsOnlySelectedBackgrounds(t *testing.T) {
	theme := map[string]config.Color{
		":selected":            {Fg: "#dcd7ba", Bg: "#363646"},
		"picker text:selected": {Fg: "#dcd7ba", Bg: "#363646"},
		"unselected":           {Fg: "#dcd7ba", Bg: "#363646"},
		"border":               {Bg: "#202020"},
		"picker border":        {Bg: "#202020"},
	}

	require.NoError(t, ApplyThemeBackgroundBlend(theme, 0.25, "", nil))

	assert.Equal(t, "#31313f", theme[":selected"].Bg)
	assert.Equal(t, "#31313f", theme["picker text:selected"].Bg)
	assert.Equal(t, "#363646", theme["unselected"].Bg)
}

func TestApplyThemeBackgroundBlend_SelectedSuffixSyntaxMatchesLegacySyntax(t *testing.T) {
	legacy := map[string]config.Color{
		"picker selected text": {Bg: "#363646"},
		"picker border":        {Bg: "#202020"},
	}
	suffix := map[string]config.Color{
		"picker text:selected": {Bg: "#363646"},
		"picker border":        {Bg: "#202020"},
	}

	require.NoError(t, ApplyThemeBackgroundBlend(legacy, 0.25, "", nil))
	require.NoError(t, ApplyThemeBackgroundBlend(suffix, 0.25, "", nil))

	assert.Equal(t, legacy["picker selected text"].Bg, suffix["picker text:selected"].Bg)
}

func TestApplyThemeBackgroundBlend_UsesEffectiveSurfaceBackground(t *testing.T) {
	for _, tt := range []struct {
		name               string
		theme              map[string]config.Color
		terminalBackground string
		want               string
	}{
		{
			name: "border background",
			theme: map[string]config.Color{
				"picker:selected": {Bg: "#808080"},
				"picker border":   {Bg: "#202020"},
			},
			want: "#707070",
		},
		{
			name: "surface preferred over border",
			theme: map[string]config.Color{
				"picker":          {Bg: "#202020"},
				"picker border":   {Bg: "#ffffff"},
				"picker:selected": {Bg: "#808080"},
			},
			terminalBackground: "#000000",
			want:               "#707070",
		},
		{
			name: "transparent surface uses terminal",
			theme: map[string]config.Color{
				"picker:selected": {Bg: "#808080"},
			},
			terminalBackground: "#202020",
			want:               "#707070",
		},
		{
			name: "default surface resolves to terminal",
			theme: map[string]config.Color{
				"picker":          {Bg: "default"},
				"picker border":   {Bg: "#ffffff"},
				"picker:selected": {Bg: "#808080"},
			},
			terminalBackground: "#202020",
			want:               "#707070",
		},
		{
			name: "missing effective background skips blending",
			theme: map[string]config.Color{
				"picker:selected": {Bg: "#808080"},
				"border":          {Fg: "#ffffff"},
			},
			want: "#808080",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, ApplyThemeBackgroundBlend(tt.theme, 0.25, tt.terminalBackground, nil))
			assert.Equal(t, tt.want, tt.theme["picker:selected"].Bg)
		})
	}
}

func TestApplyThemeBackgroundBlend_UsesTerminalPalette(t *testing.T) {
	for _, tt := range []struct {
		name            string
		theme           map[string]config.Color
		terminalPalette map[int]string
		selector        string
		want            string
	}{
		{
			name: "selected background",
			theme: map[string]config.Color{
				":selected":        {Bg: "bright black"},
				"missing:selected": {Bg: "bright red"},
				"border":           {Bg: "#202020"},
			},
			terminalPalette: map[int]string{8: "#808080"},
			selector:        ":selected",
			want:            "#707070",
		},
		{
			name: "surface border",
			theme: map[string]config.Color{
				":selected": {Bg: "#808080"},
				"border":    {Bg: "bright black"},
			},
			terminalPalette: map[int]string{8: "#202020"},
			selector:        ":selected",
			want:            "#707070",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, ApplyThemeBackgroundBlend(tt.theme, 0.25, "", tt.terminalPalette))
			assert.Equal(t, tt.want, tt.theme[tt.selector].Bg)
			if missing, ok := tt.theme["missing:selected"]; ok {
				assert.Equal(t, "bright red", missing.Bg)
			}
		})
	}
}
