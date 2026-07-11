package common

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
	switch color := parseColor(value).(type) {
	case ansi.BasicColor:
		if hex, ok := terminalPalette[int(color)]; ok {
			return hex
		}
	case ansi.IndexedColor:
		index := int(color)
		if hex, ok := terminalPalette[index]; ok {
			return hex
		}
	}
	return value
}

func blendHexColor(base, target string, ratio float64) (string, bool, error) {
	baseRGB, ok, err := parseHexRGB(base)
	if err != nil || !ok {
		return "", ok, err
	}
	targetRGB, ok, err := parseHexRGB(target)
	if err != nil || !ok {
		return "", ok, err
	}

	var out [3]uint8
	for i := range out {
		a := float64(baseRGB[i]) / 255.0
		b := float64(targetRGB[i]) / 255.0
		blended := math.Sqrt((1-ratio)*math.Pow(a, 2) + ratio*math.Pow(b, 2))
		out[i] = uint8(math.Round(blended * 255))
	}

	return fmt.Sprintf("#%02x%02x%02x", out[0], out[1], out[2]), true, nil
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
