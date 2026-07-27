package common

import (
	"strings"

	"github.com/idursun/jjui/internal/config"

	"charm.land/lipgloss/v2"
)

var DefaultPalette = NewPalette()

type Palette struct {
	styles map[string]paletteStyle
	cache  map[paletteCacheKey]lipgloss.Style
	blend  paletteBackgroundBlend
}

type paletteCacheKey struct {
	scope      string
	component  string
	role       string
	isSelected bool
}

type paletteStyle struct {
	style         lipgloss.Style
	backgroundRaw string
	backgroundSet bool
}

type paletteBackgroundBlend struct {
	ratio              float64
	terminalBackground string
	terminalPalette    map[int]string
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
		entry.backgroundRaw = color.Bg
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

func (p *Palette) ConfigureBackgroundBlend(
	ratio float64,
	terminalBackground string,
	terminalPalette map[int]string,
) {
	p.blend = paletteBackgroundBlend{
		ratio:              ratio,
		terminalBackground: terminalBackground,
		terminalPalette:    cloneTerminalPalette(terminalPalette),
	}
}

func (p *Palette) Get(scope, component, role string, isSelected bool) lipgloss.Style {
	return p.get(scope, component, role, isSelected)
}

func (p *Palette) GetBlended(scope, component, role string, isSelected bool) lipgloss.Style {
	style := p.get(scope, component, role, isSelected)
	background, backgroundSet := p.resolveBackground(paletteKeys(scope, component, role, isSelected))
	return p.applyBackgroundBlend(style, background, backgroundSet, scope, component)
}

func (p *Palette) get(scope, component, role string, isSelected bool) lipgloss.Style {
	cacheKey := paletteCacheKey{
		scope:      scope,
		component:  component,
		role:       role,
		isSelected: isSelected,
	}
	if style, ok := p.cache[cacheKey]; ok {
		return style
	}

	keys := paletteKeys(scope, component, role, isSelected)
	finalStyle := lipgloss.NewStyle()
	for _, key := range keys {
		entry, ok := p.styles[key]
		if !ok {
			continue
		}
		finalStyle = finalStyle.Inherit(entry.style)
	}
	if background, ok := p.resolveBackground(keys); ok {
		finalStyle = finalStyle.Background(parseColor(background))
	}
	p.cache[cacheKey] = finalStyle
	return finalStyle
}

func paletteKeys(scope, component, role string, isSelected bool) []string {
	baseCandidates := paletteCandidates(scope, component, role)
	keys := make([]string, 0, len(baseCandidates)*2+1)
	if isSelected {
		// Role selectors remain authoritative for selected elements. Their explicit
		// selected variants win first; broader selected selectors only fill properties
		// that the role's selected and base styles leave unset.
		contextCandidates := paletteCandidates(scope, component, "")
		contextSet := make(map[string]struct{}, len(contextCandidates))
		for _, candidate := range contextCandidates {
			contextSet[candidate] = struct{}{}
		}

		roleCandidates := make([]string, 0, len(baseCandidates)-len(contextCandidates))
		for _, candidate := range baseCandidates {
			if _, ok := contextSet[candidate]; !ok {
				roleCandidates = append(roleCandidates, candidate)
			}
		}

		for _, candidate := range roleCandidates {
			keys = append(keys, candidate+":"+string(config.SelectedVariant))
		}
		keys = append(keys, roleCandidates...)
		for _, candidate := range contextCandidates {
			keys = append(keys, candidate+":"+string(config.SelectedVariant))
		}
		keys = append(keys, ":"+string(config.SelectedVariant))
		keys = append(keys, contextCandidates...)
	} else {
		keys = append(keys, baseCandidates...)
	}
	return keys
}

func (p *Palette) resolveBackground(keys []string) (string, bool) {
	for _, key := range keys {
		entry, ok := p.styles[key]
		if ok && entry.backgroundSet {
			return entry.backgroundRaw, true
		}
	}
	return "", false
}

func (p *Palette) applyBackgroundBlend(
	style lipgloss.Style,
	background string,
	backgroundSet bool,
	scope string,
	component string,
) lipgloss.Style {
	if p.blend.ratio == 0 || !backgroundSet {
		return style
	}

	target := p.backgroundBlendTarget(scope, component)
	if target == "" {
		return style
	}
	base := resolvePaletteColor(background, p.blend.terminalPalette)
	blended, ok, err := blendHexColor(base, target, p.blend.ratio)
	if err != nil || !ok {
		return style
	}

	return style.Background(parseColor(blended))
}

func (p *Palette) backgroundBlendTarget(scope, component string) string {
	if background, ok := p.resolveBackground(paletteKeys(scope, component, "", false)); ok {
		return resolveBlendTarget(background, p.blend.terminalBackground, p.blend.terminalPalette)
	}

	if background, ok := p.resolveBackground(paletteKeys(scope, component, "border", false)); ok {
		return resolveBlendTarget(background, p.blend.terminalBackground, p.blend.terminalPalette)
	}
	return p.blend.terminalBackground
}

func cloneTerminalPalette(terminalPalette map[int]string) map[int]string {
	if terminalPalette == nil {
		return nil
	}
	cloned := make(map[int]string, len(terminalPalette))
	for index, value := range terminalPalette {
		cloned[index] = value
	}
	return cloned
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

func resolveBlendTarget(value, terminalBackground string, terminalPalette map[int]string) string {
	if value == "default" {
		return terminalBackground
	}
	return resolvePaletteColor(value, terminalPalette)
}
