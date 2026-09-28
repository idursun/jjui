package render

import (
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/theme"
)

type contrastPair struct {
	foreground ansi.Color
	background ansi.Color
}

// HighlightContrast is shared by the visible rows for one frame. Cache colour
// pairs so contrast searches are per style, not per character. A fresh cache
// also avoids retaining stale results after theme or terminal palette changes.
type HighlightContrast struct {
	surface         theme.RGB
	terminalPalette map[int]string
	pairs           map[contrastPair]contrastPair
}

// NewHighlightContrast softens highlight backgrounds before correcting foregrounds.
func NewHighlightContrast(surface ansi.Color, terminalBackground string, terminalPalette map[int]string, dark bool) *HighlightContrast {
	a := &HighlightContrast{terminalPalette: terminalPalette, pairs: make(map[contrastPair]contrastPair)}
	a.surface = theme.ResolveSurface(surface, terminalBackground, terminalPalette, dark)
	return a
}

func (a *HighlightContrast) rgb(value ansi.Color) theme.RGB {
	c, _ := theme.ResolveRGBFallback(value, a.terminalPalette)
	return c
}

func (a *HighlightContrast) adjust(pair contrastPair) contrastPair {
	if result, ok := a.pairs[pair]; ok {
		return result
	}
	result := pair
	bg := a.rgb(pair.background)
	adjustedBG := theme.HighlightBackground(bg, a.surface)
	if adjustedBG != bg {
		result.background = adjustedBG.Color()
	}
	if pair.foreground != nil {
		fg := a.rgb(pair.foreground)
		adjustedFG := theme.Foreground(fg, adjustedBG)
		if adjustedFG != fg {
			result.foreground = adjustedFG.Color()
		}
	}
	a.pairs[pair] = result
	return result
}

type highlightContrastEffect struct {
	rect     layout.Rectangle
	adjuster *HighlightContrast
	z        int
}

func (e highlightContrastEffect) GetZ() int { return e.z }

// Apply runs after word, line and selection backgrounds have been composed.
// Work on screen cells so ANSI resets and token boundaries cannot undo it.
func (e highlightContrastEffect) Apply(buf uv.Screen) {
	applyContrast(buf, e.rect, e.adjuster.adjust)
}

// applyContrast handles visible colour order and cell preservation.
func applyContrast(buf uv.Screen, rect layout.Rectangle, adjust func(contrastPair) contrastPair) {
	iterateCells(buf, rect, func(cell *uv.Cell) *uv.Cell {
		if cell == nil || cell.Width == 0 {
			return nil
		}
		pair := contrastPair{cell.Style.Fg, cell.Style.Bg}
		reverse := cell.Style.Attrs&uv.AttrReverse != 0
		if reverse {
			pair.foreground, pair.background = pair.background, pair.foreground
		}
		switch pair.background.(type) {
		case nil, lipgloss.NoColor:
			return nil
		}
		adjusted := adjust(pair)
		if adjusted == pair {
			return nil
		}
		updated := cell.Clone()
		if reverse {
			adjusted.foreground, adjusted.background = adjusted.background, adjusted.foreground
		}
		updated.Style.Fg, updated.Style.Bg = adjusted.foreground, adjusted.background
		return updated
	})
}

// Add applies highlight contrast after its background layers.
func (a *HighlightContrast) Add(dl *DisplayContext, rect layout.Rectangle, z int) {
	dl.AddEffect(highlightContrastEffect{rect: rect, adjuster: a, z: z})
}
