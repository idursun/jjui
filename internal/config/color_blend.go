package config

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

func applyThemeBackgroundBlend(theme map[string]Color, ratio float64, terminalBackground string, terminalPalette map[int]string) error {
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

func blendTargetColor(theme map[string]Color, selectedFields []string, terminalBackground string, terminalPalette map[int]string) string {
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

func resolvedBackground(theme map[string]Color, fields []string) string {
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
	if index, ok := ParseANSIColorIndex(value); ok {
		if hex, ok := terminalPalette[index]; ok {
			return hex
		}
	}
	return value
}

// ParseANSIColorIndex resolves a named, numeric, or ansi-color-prefixed value.
func ParseANSIColorIndex(value string) (int, bool) {
	switch value {
	case "black":
		return 0, true
	case "red":
		return 1, true
	case "green":
		return 2, true
	case "yellow":
		return 3, true
	case "blue":
		return 4, true
	case "magenta":
		return 5, true
	case "cyan":
		return 6, true
	case "white":
		return 7, true
	case "bright black":
		return 8, true
	case "bright red":
		return 9, true
	case "bright green":
		return 10, true
	case "bright yellow":
		return 11, true
	case "bright blue":
		return 12, true
	case "bright magenta":
		return 13, true
	case "bright cyan":
		return 14, true
	case "bright white":
		return 15, true
	}

	if after, ok := strings.CutPrefix(value, "ansi-color-"); ok {
		value = after
	}
	index, err := strconv.Atoi(value)
	if err != nil || index < 0 || index > 255 {
		return 0, false
	}
	return index, true
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

	for i := range rgb {
		channel, err := strconv.ParseUint(value[1+i*2:3+i*2], 16, 8)
		if err != nil {
			return rgb, true, fmt.Errorf("invalid hex color %q", value)
		}
		rgb[i] = uint8(channel)
	}
	return rgb, true, nil
}
