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
	cache  map[paletteCacheKey]lipgloss.Style
}

type paletteCacheKey struct {
	scope      string
	component  string
	role       string
	isSelected bool
}

type paletteStyle struct {
	style         lipgloss.Style
	background    color.Color
	backgroundSet bool
}

func NewPalette() *Palette {
	return &Palette{
		styles: make(map[string]paletteStyle),
		cache:  make(map[paletteCacheKey]lipgloss.Style),
	}
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

func (p *Palette) Get(scope, component, role string, isSelected bool) lipgloss.Style {
	cacheKey := paletteCacheKey{
		scope:      scope,
		component:  component,
		role:       role,
		isSelected: isSelected,
	}
	if style, ok := p.cache[cacheKey]; ok {
		return style
	}

	baseCandidates := paletteCandidates(scope, component, role)
	keys := make([]string, 0, len(baseCandidates)*2+1)
	if isSelected {
		for _, candidate := range baseCandidates {
			keys = append(keys, candidate+":"+string(config.SelectedVariant))
		}
		keys = append(keys, ":"+string(config.SelectedVariant))
	}
	keys = append(keys, baseCandidates...)

	finalStyle := lipgloss.NewStyle()
	var background color.Color
	backgroundSet := false
	for _, key := range keys {
		entry, ok := p.styles[key]
		if !ok {
			continue
		}
		finalStyle = finalStyle.Inherit(entry.style)
		if !backgroundSet && entry.backgroundSet {
			background = entry.background
			backgroundSet = true
		}
	}
	if backgroundSet {
		finalStyle = finalStyle.Background(background)
	}
	p.cache[cacheKey] = finalStyle
	return finalStyle
}

func paletteCandidates(scope, component, role string) []string {
	candidates := make([]string, 0, 7)
	seen := make(map[string]struct{}, 7)
	add := func(parts ...string) {
		nonEmpty := parts[:0]
		for _, part := range parts {
			if part != "" {
				nonEmpty = append(nonEmpty, part)
			}
		}
		if len(nonEmpty) == 0 {
			return
		}
		key := strings.Join(nonEmpty, " ")
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, key)
	}

	add(scope, component, role)
	add(scope, component)
	add(scope, role)
	add(scope)
	add(component, role)
	add(component)
	add(role)
	return candidates
}

func (p *Palette) GetBorder(scope, component, role string, isSelected bool, border lipgloss.Border) lipgloss.Style {
	style := p.Get(scope, component, role, isSelected)
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
