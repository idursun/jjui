package theme

import (
	"fmt"
	"image/color"
	"math"
	"slices"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var ansiColorNames = [...]string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"}

func parseColor(value string) color.Color {
	if value == "" || value == "default" {
		return lipgloss.NoColor{}
	}
	if len(value) == 7 && value[0] == '#' {
		return lipgloss.Color(value)
	}

	baseName := value
	brightOffset := 0
	if after, ok := strings.CutPrefix(value, "bright "); ok {
		baseName = after
		brightOffset = 8
	}
	if index := slices.Index(ansiColorNames[:], baseName); index >= 0 {
		return ansi.BasicColor(index + brightOffset)
	}

	if after, ok := strings.CutPrefix(value, "ansi-color-"); ok {
		value = after
	}
	if index, err := strconv.Atoi(value); err == nil && index >= 0 && index <= 255 {
		return lipgloss.Color(value)
	}
	return lipgloss.NoColor{}
}

func resolvePaletteColor(value string, terminalPalette map[int]string) string {
	if resolved, ok := ResolveTerminalColor(parseColor(value), terminalPalette); ok {
		return resolved
	}
	return value
}

// ResolveTerminalColor returns a hex RGB colour, using the reported terminal
// palette for ANSI colours. Unset colours and unreported slots are unresolved.
func ResolveTerminalColor(value color.Color, terminalPalette map[int]string) (string, bool) {
	rgb, ok := ResolveRGB(value, terminalPalette)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2]), true
}

func parseHexRGB(value string) ([3]uint8, bool, error) {
	var rgb [3]uint8
	if len(value) != 7 || value[0] != '#' {
		return rgb, false, nil
	}
	parsed := ansi.XParseColor(value)
	if parsed == nil {
		return rgb, true, fmt.Errorf("invalid hex color %q", value)
	}
	r, g, b, _ := parsed.RGBA()
	return [3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}, true, nil
}

// ResolveRGB resolves known terminal colours. Unreported ANSI slots remain unknown.
func ResolveRGB(value color.Color, terminalPalette map[int]string) (RGB, bool) {
	switch v := value.(type) {
	case nil, lipgloss.NoColor:
		return RGB{}, false
	case ansi.BasicColor:
		hex, ok := terminalPalette[int(v)]
		if !ok {
			return RGB{}, false
		}
		value = ansi.XParseColor(hex)
	case ansi.IndexedColor:
		hex, ok := terminalPalette[int(v)]
		if !ok {
			return RGB{}, false
		}
		value = ansi.XParseColor(hex)
	}
	if value == nil {
		return RGB{}, false
	}
	r, g, b, _ := value.RGBA()
	return RGB{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}, true
}

// ResolveRGBFallback uses standard ANSI values when the terminal has not reported them.
// This is the fallback used by syntax highlighting; unset colours remain unknown.
func ResolveRGBFallback(value color.Color, terminalPalette map[int]string) (RGB, bool) {
	if rgb, ok := ResolveRGB(value, terminalPalette); ok {
		return rgb, true
	}
	switch value.(type) {
	case nil, lipgloss.NoColor:
		return RGB{}, false
	}
	r, g, b, _ := value.RGBA()
	return RGB{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}, true
}

// ResolveRGB uses this palette's reported terminal colours without guessing defaults.
func (p *Palette) ResolveRGB(value color.Color) (RGB, bool) {
	return ResolveRGB(value, p.blend.terminalPalette)
}

// ResolveSurface prefers the theme surface, then the terminal background, then mode.
func ResolveSurface(surface color.Color, terminalBackground string, terminalPalette map[int]string, dark bool) RGB {
	if rgb, ok := ResolveRGBFallback(surface, terminalPalette); ok {
		return rgb
	}
	if terminalBackground != "" {
		if rgb, ok := ResolveRGBFallback(lipgloss.Color(terminalBackground), terminalPalette); ok {
			return rgb
		}
	}
	if dark {
		return RGB{}
	}
	return RGB{255, 255, 255}
}

// RGB is an opaque, resolved sRGB colour.
type RGB [3]uint8

func (c RGB) mix(target RGB, amount float64) RGB {
	var result RGB
	for i := range result {
		result[i] = uint8(math.Round(float64(c[i]) + (float64(target[i])-float64(c[i]))*amount))
	}
	return result
}

func (c RGB) Color() ansi.RGBColor { return ansi.RGBColor{R: c[0], G: c[1], B: c[2]} }
