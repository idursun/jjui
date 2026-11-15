package theme

import (
	"fmt"
	"math"

	"charm.land/lipgloss/v2"
)

type paletteBackgroundBlend struct {
	ratio              float64
	terminalBackground string
	terminalPalette    map[int]string
}

func (p *Palette) SetBackgroundBlend(ratio float64) { p.blend.ratio = ratio }

func (p *Palette) BackgroundBlend() float64 { return p.blend.ratio }

func (p *Palette) SetTerminalBackground(hex string) { p.blend.terminalBackground = hex }

func (p *Palette) TerminalBackground() string { return p.blend.terminalBackground }

func (p *Palette) SetTerminalPaletteColor(slot int, hex string) {
	if p.blend.terminalPalette == nil {
		p.blend.terminalPalette = make(map[int]string)
	}
	p.blend.terminalPalette[slot] = hex
}

// TerminalPalette returns the live map; callers must not mutate it.
func (p *Palette) TerminalPalette() map[int]string { return p.blend.terminalPalette }

func (p *Palette) SetDark(dark bool) { p.dark = dark }

func (p *Palette) IsDark() bool { return p.dark }

func (p *Palette) GetBlended(scope, component, role string, isSelected bool) lipgloss.Style {
	return p.GetBlendedCustom(scope, component, role, isSelected, p.blend.ratio)
}

func (p *Palette) GetBlendedCustom(
	scope, component, role string,
	isSelected bool,
	ratio float64,
) lipgloss.Style {
	style := p.get(scope, component, role, isSelected)
	background, backgroundSet := p.resolveBackground(paletteKeys(scope, component, role, isSelected))
	return p.applyBackgroundBlend(style, background, backgroundSet, scope, component, ratio)
}

func (p *Palette) BlendBackgroundCustom(
	style lipgloss.Style,
	scope, component string,
	ratio float64,
) lipgloss.Style {
	background, ok := ResolveTerminalColor(style.GetBackground(), p.blend.terminalPalette)
	if !ok {
		return style
	}
	return p.applyBackgroundBlend(
		style,
		background,
		true,
		scope,
		component,
		ratio,
	)
}

func (p *Palette) applyBackgroundBlend(
	style lipgloss.Style,
	background string,
	backgroundSet bool,
	scope string,
	component string,
	ratio float64,
) lipgloss.Style {
	if ratio == 0 || !backgroundSet {
		return style
	}

	target := p.backgroundBlendTarget(scope, component)
	if target == "" {
		return style
	}
	base := resolvePaletteColor(background, p.blend.terminalPalette)
	blended, ok, err := blendHexColor(base, target, ratio)
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

func resolveBlendTarget(value, terminalBackground string, terminalPalette map[int]string) string {
	if value == "default" {
		return terminalBackground
	}
	return resolvePaletteColor(value, terminalPalette)
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
