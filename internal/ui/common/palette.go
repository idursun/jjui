package common

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/idursun/jjui/internal/config"

	"charm.land/lipgloss/v2"
)

var DefaultPalette = NewPalette()

type Palette struct {
	styles map[string]paletteStyle
	cache  map[string]lipgloss.Style
}

type paletteStyle struct {
	style         lipgloss.Style
	background    color.Color
	backgroundSet bool
}

func NewPalette() *Palette {
	return &Palette{
		styles: make(map[string]paletteStyle),
		cache:  make(map[string]lipgloss.Style),
	}
}

func (p *Palette) add(key string, style lipgloss.Style) {
	entry := paletteStyle{style: style}
	if _, noColor := style.GetBackground().(lipgloss.NoColor); !noColor {
		entry.background = style.GetBackground()
		entry.backgroundSet = true
	}
	p.styles[config.ParseColorSelector(key).Key()] = entry
}

func (p *Palette) addColor(key string, color config.Color) {
	style := createStyleFrom(color)
	entry := paletteStyle{style: style}
	if color.Bg != "" {
		entry.background = parseColor(color.Bg)
		entry.backgroundSet = true
	}
	p.styles[config.ParseColorSelector(key).Key()] = entry
}

func (p *Palette) get(fields ...string) lipgloss.Style {
	key := config.ParseColorSelector(strings.Join(fields, " ")).Key()
	if entry, ok := p.styles[key]; ok {
		return entry.style
	}
	return lipgloss.NewStyle()
}

func (p *Palette) getBackground(fields ...string) (color.Color, bool) {
	key := config.ParseColorSelector(strings.Join(fields, " ")).Key()
	entry, ok := p.styles[key]
	return entry.background, ok && entry.backgroundSet
}

func (p *Palette) Update(styleMap map[string]config.Color) {
	clear(p.cache)
	clear(p.styles)
	normalizedStyles := config.NormalizeColorSelectors(styleMap)
	for key, color := range normalizedStyles {
		p.addColor(key, color)
	}

	if color, ok := normalizedStyles["diff added"]; ok {
		p.addColor("added", color)
	}
	if color, ok := normalizedStyles["diff renamed"]; ok {
		p.addColor("renamed", color)
	}
	if color, ok := normalizedStyles["diff copied"]; ok {
		p.addColor("copied", color)
	}
	if color, ok := normalizedStyles["diff modified"]; ok {
		p.addColor("modified", color)
	}
	if color, ok := normalizedStyles["diff removed"]; ok {
		p.addColor("deleted", color)
	}
}

func (p *Palette) Get(selector string) lipgloss.Style {
	parsed := config.ParseColorSelector(selector)
	cacheKey := parsed.Key()
	if style, ok := p.cache[cacheKey]; ok {
		return style
	}
	fields := parsed.LegacyFields()
	length := len(fields)

	finalStyle := lipgloss.NewStyle()
	var deferredSelectedStyles [][]string
	deferSelectedStyles := hasSelectedSubstyle(fields)
	var selectedRoleBackground color.Color
	selectedRoleBackgroundSet := false
	// for a selector like "a b c", we want to inherit styles from the most specific to the least specific
	// first pass: "a b c", "a b", "a"
	// second pass: "b c", "b"
	// third pass: "c"
	start := 0
	for start < length {
		for end := length; end > start; end-- {
			selectorFields := fields[start:end]
			if deferSelectedStyles && selectorFields[len(selectorFields)-1] == "selected" {
				deferredSelectedStyles = append(deferredSelectedStyles, selectorFields)
				continue
			}
			style := p.get(selectorFields...)
			if hasSelectedSubstyle(selectorFields) && !selectedRoleBackgroundSet {
				selectedRoleBackground, selectedRoleBackgroundSet = p.getBackground(selectorFields...)
			}
			finalStyle = finalStyle.Inherit(style)
			if semanticFields := withoutSelectedModifier(selectorFields); len(semanticFields) > 0 {
				finalStyle = finalStyle.Inherit(p.get(semanticFields...))
			}
		}
		start++
	}
	var selectedBackground color.Color
	selectedBackgroundSet := false
	for _, selectorFields := range deferredSelectedStyles {
		style := p.get(selectorFields...)
		if !selectedBackgroundSet {
			selectedBackground, selectedBackgroundSet = p.getBackground(selectorFields...)
		}
		finalStyle = finalStyle.Inherit(style)
	}
	if selectedRoleBackgroundSet {
		finalStyle = finalStyle.Background(selectedRoleBackground)
	} else if selectedBackgroundSet {
		finalStyle = finalStyle.Background(selectedBackground)
	}
	p.cache[cacheKey] = finalStyle
	return finalStyle
}

func (p *Palette) GetVariant(selector string, variant config.ColorVariant, enabled bool) lipgloss.Style {
	if !enabled {
		return p.Get(selector)
	}
	return p.Get(selector + ":" + string(variant))
}

func hasSelectedSubstyle(fields []string) bool {
	for i, field := range fields {
		if field == "selected" && i < len(fields)-1 {
			return true
		}
	}
	return false
}

func withoutSelectedModifier(fields []string) []string {
	for i, field := range fields {
		if field != "selected" || i == len(fields)-1 {
			continue
		}
		result := make([]string, 0, len(fields)-1)
		result = append(result, fields[:i]...)
		result = append(result, fields[i+1:]...)
		return result
	}
	return nil
}

func (p *Palette) GetBorder(selector string, border lipgloss.Border) lipgloss.Style {
	style := p.Get(selector)
	return lipgloss.NewStyle().
		Border(border).
		Foreground(style.GetForeground()).
		Background(style.GetBackground()).
		BorderForeground(style.GetForeground()).
		BorderBackground(style.GetBackground())
}

func createStyleFrom(color config.Color) lipgloss.Style {
	style := lipgloss.NewStyle()
	if color.Fg != "" {
		style = style.Foreground(parseColor(color.Fg))
	}
	if color.Bg != "" {
		style = style.Background(parseColor(color.Bg))
	}

	if color.Bold != nil {
		style = style.Bold(*color.Bold)
	}
	if color.Italic != nil {
		style = style.Italic(*color.Italic)
	}
	if color.Underline != nil {
		style = style.Underline(*color.Underline)
	}
	if color.Strikethrough != nil {
		style = style.Strikethrough(*color.Strikethrough)
	}
	if color.Reverse != nil {
		style = style.Reverse(*color.Reverse)
	}

	return style
}

func ApplyThemeBackgroundBlend(theme map[string]config.Color, ratio float64, terminalBackground string, terminalPalette map[int]string) error {
	if ratio == 0 {
		return nil
	}

	for key, color := range theme {
		selector := config.ParseColorSelector(key)
		if color.Bg == "" || !selector.HasVariant(config.SelectedVariant) {
			continue
		}
		fields := selector.Fields()
		target := blendTargetColor(theme, fields, terminalBackground, terminalPalette)
		if target == "" {
			continue
		}
		base := resolvePaletteColor(color.Bg, terminalPalette)
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

func blendTargetColor(theme map[string]config.Color, fields []string, terminalBackground string, terminalPalette map[int]string) string {
	if bg := resolvedBackground(theme, fields); bg != "" {
		return resolveBlendTarget(bg, terminalBackground, terminalPalette)
	}

	if bg := resolvedBorderBackground(theme, fields); bg != "" {
		return resolveBlendTarget(bg, terminalBackground, terminalPalette)
	}
	return terminalBackground
}

func resolveBlendTarget(value, terminalBackground string, terminalPalette map[int]string) string {
	if value == "default" {
		return terminalBackground
	}
	return resolvePaletteColor(value, terminalPalette)
}

func resolvedBorderBackground(theme map[string]config.Color, fields []string) string {
	for start := 0; start < len(fields); start++ {
		for end := len(fields); end > start; end-- {
			key := strings.Join(append(append([]string(nil), fields[start:end]...), "border"), " ")
			if color, ok := theme[key]; ok && color.Bg != "" {
				return color.Bg
			}
		}
	}
	if color, ok := theme["border"]; ok {
		return color.Bg
	}
	return ""
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
