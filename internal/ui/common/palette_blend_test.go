package common

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestPaletteGetBlended_LeavesRawStyleUnchanged(t *testing.T) {
	palette := NewPalette()
	palette.Update(map[string]config.Color{
		"picker:selected": {Bg: "#363646"},
		"picker border":   {Bg: "#202020"},
	})
	palette.ConfigureBackgroundBlend(
		0.25,
		"",
		nil,
	)

	assert.Equal(t, lipgloss.Color("#363646"), palette.Get("picker", "", "", true).GetBackground())
	assert.Equal(t, lipgloss.Color("#31313f"), palette.GetBlended("picker", "", "", true).GetBackground())
	assert.Equal(t, lipgloss.Color("#363646"), palette.Get("picker", "", "", true).GetBackground())
}

func TestPaletteGetBlended_CanBlendNonSelectedStyle(t *testing.T) {
	palette := NewPalette()
	palette.Update(map[string]config.Color{
		"picker":       {Bg: "#202020"},
		"picker badge": {Bg: "#808080"},
	})
	palette.ConfigureBackgroundBlend(
		0.25,
		"",
		nil,
	)

	assert.Equal(t, lipgloss.Color("#808080"), palette.Get("picker", "", "badge", false).GetBackground())
	assert.Equal(t, lipgloss.Color("#707070"), palette.GetBlended("picker", "", "badge", false).GetBackground())
}

func TestPaletteGetBlendedCustom_UsesProvidedRatio(t *testing.T) {
	palette := NewPalette()
	palette.Update(map[string]config.Color{
		"picker badge": {Bg: "#808080"},
	})
	palette.ConfigureBackgroundBlend(0.25, "#000000", nil)

	assert.Equal(
		t,
		lipgloss.Color("#404040"),
		palette.GetBlendedCustom("picker", "", "badge", false, 0.75).GetBackground(),
	)
}

func TestPaletteBlendBackgroundCustom_UsesProvidedStyle(t *testing.T) {
	palette := NewPalette()
	palette.Update(map[string]config.Color{
		"annotation": {Fg: "#ff0000"},
	})
	palette.ConfigureBackgroundBlend(0.25, "#000000", nil)

	source := lipgloss.NewStyle().Background(lipgloss.Color("#00aa00"))
	style := palette.BlendBackgroundCustom(source, "", "", 0.7)

	assert.Equal(t, lipgloss.Color("#005d00"), style.GetBackground())
	assert.IsType(t, lipgloss.NoColor{}, style.GetForeground())
	assert.Equal(t, lipgloss.Color("#00aa00"), source.GetBackground())
}

func TestPaletteBlendBackgroundCustom_ResolvesTerminalPalette(t *testing.T) {
	palette := NewPalette()
	palette.ConfigureBackgroundBlend(0.25, "#000000", map[int]string{2: "#00aa00"})

	source := lipgloss.NewStyle().Background(lipgloss.Color("2"))
	style := palette.BlendBackgroundCustom(source, "annotation", "", 0.7)

	assert.Equal(t, lipgloss.Color("#005d00"), style.GetBackground())
}

func TestPaletteGetBlended_SelectedSuffixSyntaxMatchesLegacySyntax(t *testing.T) {
	legacy := NewPalette()
	legacy.Update(map[string]config.Color{
		"picker selected text": {Bg: "#363646"},
		"picker border":        {Bg: "#202020"},
	})
	legacy.ConfigureBackgroundBlend(
		0.25,
		"",
		nil,
	)
	suffix := NewPalette()
	suffix.Update(map[string]config.Color{
		"picker text:selected": {Bg: "#363646"},
		"picker border":        {Bg: "#202020"},
	})
	suffix.ConfigureBackgroundBlend(
		0.25,
		"",
		nil,
	)

	assert.Equal(
		t,
		legacy.GetBlended("picker", "", "text", true),
		suffix.GetBlended("picker", "", "text", true),
	)
}

func TestPaletteGetBlended_UsesContainingSurfaceBackground(t *testing.T) {
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
			palette := NewPalette()
			palette.Update(tt.theme)
			palette.ConfigureBackgroundBlend(0.25, tt.terminalBackground, nil)

			assert.Equal(t, lipgloss.Color(tt.want), palette.GetBlended("picker", "", "", true).GetBackground())
		})
	}
}

func TestPaletteGetBlended_UsesTerminalPalette(t *testing.T) {
	for _, tt := range []struct {
		name            string
		theme           map[string]config.Color
		terminalPalette map[int]string
		want            string
	}{
		{
			name: "style background",
			theme: map[string]config.Color{
				":selected": {Bg: "bright black"},
				"border":    {Bg: "#202020"},
			},
			terminalPalette: map[int]string{8: "#808080"},
			want:            "#707070",
		},
		{
			name: "border background",
			theme: map[string]config.Color{
				":selected": {Bg: "#808080"},
				"border":    {Bg: "bright black"},
			},
			terminalPalette: map[int]string{8: "#202020"},
			want:            "#707070",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			palette := NewPalette()
			palette.Update(tt.theme)
			palette.ConfigureBackgroundBlend(0.25, "", tt.terminalPalette)

			assert.Equal(t, lipgloss.Color(tt.want), palette.GetBlended("", "", "", true).GetBackground())
		})
	}
}
