package common

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseColor(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  color.Color
	}{
		{name: "hex", value: "#ff0000", want: lipgloss.Color("#ff0000")},
		{name: "ANSI index", value: "123", want: lipgloss.Color("123")},
		{name: "named ANSI colour", value: "red", want: lipgloss.Color("1")},
		{name: "bright named ANSI colour", value: "bright blue", want: lipgloss.Color("12")},
		{name: "ANSI colour prefix", value: "ansi-color-42", want: lipgloss.Color("42")},
		{name: "default", value: "default", want: lipgloss.NoColor{}},
		{name: "invalid name", value: "not-a-color", want: lipgloss.NoColor{}},
		{name: "out-of-range ANSI index", value: "300", want: lipgloss.NoColor{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseColor(tt.value))
		})
	}
}

func TestResolvePaletteColor(t *testing.T) {
	terminalPalette := map[int]string{1: "#cc0000", 42: "#00cc00"}

	for _, tt := range []struct {
		name  string
		value string
		want  string
	}{
		{name: "named ANSI colour", value: "red", want: "#cc0000"},
		{name: "indexed ANSI colour", value: "ansi-color-42", want: "#00cc00"},
		{name: "missing palette entry", value: "bright red", want: "bright red"},
		{name: "hex colour", value: "#123456", want: "#123456"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolvePaletteColor(tt.value, terminalPalette))
		})
	}
}

func TestBlendHexColor_UsesGammaCorrectRGBBlend(t *testing.T) {
	got, ok, err := blendHexColor("#ff0000", "#000000", 0.5)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "#b40000", got)
}
