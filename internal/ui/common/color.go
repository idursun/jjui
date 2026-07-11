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
	"github.com/idursun/jjui/internal/config"
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

func ApplyThemeBackgroundBlend(theme map[string]config.Color, ratio float64, terminalBackground string, terminalPalette map[int]string) error {
	if ratio == 0 {
		return nil
	}

	for key, color := range theme {
		fields := strings.Fields(key)
		if color.Bg == "" || !slices.Contains(fields, "selected") {
			continue
		}
		target := blendTargetColor(theme, fields, terminalBackground, terminalPalette)
		if target == "" {
			continue
		}
		base := blendBaseColor(color.Bg, terminalPalette)
		blended, ok, err := blendHexColor(base, target, ratio)
		if err != nil {
			return fmt.Errorf("%q bg: %w", key, err)
		}
		if !ok {
			continue
		}
		color.Bg = blended
		theme[key] = color
	}
	return nil
}

func blendTargetColor(theme map[string]config.Color, selectedFields []string, terminalBackground string, terminalPalette map[int]string) string {
	if bg := resolvedBackground(theme, withoutSelectedField(selectedFields)); bg != "" {
		return resolveBlendTarget(bg, terminalBackground, terminalPalette)
	}

	borderFields := surfaceBorderFields(selectedFields)
	if bg := resolvedBackground(theme, borderFields); bg != "" {
		return resolveBlendTarget(bg, terminalBackground, terminalPalette)
	}
	return terminalBackground
}

func withoutSelectedField(fields []string) []string {
	selectedIndex := slices.Index(fields, "selected")
	if selectedIndex < 0 {
		return nil
	}
	result := make([]string, 0, len(fields)-1)
	result = append(result, fields[:selectedIndex]...)
	result = append(result, fields[selectedIndex+1:]...)
	return result
}

func resolveBlendTarget(value, terminalBackground string, terminalPalette map[int]string) string {
	if value == "default" {
		return terminalBackground
	}
	return blendBaseColor(value, terminalPalette)
}

func surfaceBorderFields(selectedFields []string) []string {
	selectedIndex := slices.Index(selectedFields, "selected")
	if selectedIndex < 0 {
		return nil
	}
	borderFields := make([]string, 0, selectedIndex+1)
	borderFields = append(borderFields, selectedFields[:selectedIndex]...)
	borderFields = append(borderFields, "border")
	return borderFields
}

func resolvedBackground(theme map[string]config.Color, fields []string) string {
	length := len(fields)
	start := 0
	for start < length {
		for end := length; end > start; end-- {
			if color, ok := theme[strings.Join(fields[start:end], " ")]; ok && color.Bg != "" {
				return color.Bg
			}
		}
		start++
	}
	return ""
}

func blendBaseColor(value string, terminalPalette map[int]string) string {
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
