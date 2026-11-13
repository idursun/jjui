package render

import (
	"testing"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestContrastEffectPreservesGraphemesAndAttributes(t *testing.T) {
	buffer := uv.NewScreenBuffer(12, 1)
	rect := layout.Rect(0, 0, 12, 1)
	text := lipgloss.NewStyle().Foreground(lipgloss.Color("#b695f3")).Background(lipgloss.Color("#4f8b57")).Bold(true).Italic(true).Underline(true).Render("界e\u0301🙂x")
	uv.NewStyledString(text).Draw(buffer, rect)
	before := make([]*uv.Cell, 12)
	for x := range 12 {
		before[x] = buffer.CellAt(x, 0).Clone()
	}
	adjuster := NewHighlightContrast(nil, "#282a35", nil, true)
	effect := highlightContrastEffect{rect: rect, adjuster: adjuster}
	effect.Apply(buffer)
	for x := range 12 {
		got := buffer.CellAt(x, 0)
		assert.Equal(t, before[x].Content, got.Content, "x=%d", x)
		assert.Equal(t, before[x].Width, got.Width, "x=%d", x)
		assert.Equal(t, before[x].Style.Attrs, got.Style.Attrs)
		assert.Equal(t, before[x].Style.Underline, got.Style.Underline)
		if got.Width > 0 && got.Style.Fg != nil && got.Style.Bg != nil {
			assert.GreaterOrEqual(t, theme.Ratio(adjuster.rgb(got.Style.Fg), adjuster.rgb(got.Style.Bg)), theme.Minimum)
		}
	}
	assert.Len(t, adjuster.pairs, 1, "one calculation per colour pair, not per cell")
}

func TestContrastEffectClipsAndLeavesUnknownBackgroundAlone(t *testing.T) {
	buffer := uv.NewScreenBuffer(3, 1)
	uv.NewStyledString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("abc")).Draw(buffer, layout.Rect(0, 0, 3, 1))
	before := buffer.CellAt(1, 0).Clone()
	effect := highlightContrastEffect{rect: layout.Rect(-2, -2, 10, 10), adjuster: NewHighlightContrast(nil, "", nil, true)}
	effect.Apply(buffer)
	assert.Equal(t, before, buffer.CellAt(1, 0))
}

func TestContrastEffectUsesReportedANSIColours(t *testing.T) {
	adjuster := NewHighlightContrast(nil, "#282a35", map[int]string{5: "#b695f3", 2: "#4f8b57"}, true)
	pair := contrastPair{ansi.Magenta, ansi.Green}
	got := adjuster.adjust(pair)
	assert.GreaterOrEqual(t, theme.Ratio(adjuster.rgb(got.foreground), adjuster.rgb(got.background)), theme.Minimum)
	assert.NotEqual(t, pair, got)
	// A theme/palette reload starts a new frame with no old resolved pairs.
	reloaded := NewHighlightContrast(nil, "#282a35", map[int]string{5: "#ffffff", 2: "#000000"}, true)
	assert.Equal(t, pair, reloaded.adjust(pair))
}

func TestContrastBackgroundUsesSurfacePolarity(t *testing.T) {
	for _, tt := range []struct {
		name                      string
		surface, background, edge theme.RGB
	}{
		{"dark", theme.RGB{40, 42, 53}, theme.RGB{79, 139, 87}, theme.RGB{255, 255, 255}},
		{"light", theme.RGB{250, 250, 250}, theme.RGB{80, 130, 80}, theme.RGB{0, 0, 0}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			adjusted := theme.HighlightBackground(tt.background, tt.surface)
			assert.GreaterOrEqual(t, theme.Ratio(adjusted, tt.edge), theme.HighlightTextContrast)
			// One uniform background change for the entire highlight, even for spaces.
			adjuster := &HighlightContrast{surface: tt.surface, pairs: make(map[contrastPair]contrastPair)}
			blank := adjuster.adjust(contrastPair{background: tt.background.Color()})
			text := adjuster.adjust(contrastPair{foreground: theme.RGB{120, 100, 180}.Color(), background: tt.background.Color()})
			assert.Equal(t, blank.background, text.background)
			assert.Nil(t, blank.foreground)
		})
	}
	// Respect an explicit annotation surface, even when the terminal uses the
	// opposite mode; transparent/default surfaces follow the terminal instead.
	explicit := NewHighlightContrast(lipgloss.Color("#fafafa"), "#282a35", nil, true)
	assert.Equal(t, theme.RGB{250, 250, 250}, explicit.surface)
	transparent := NewHighlightContrast(lipgloss.NoColor{}, "#282a35", nil, false)
	assert.Equal(t, theme.RGB{40, 42, 53}, transparent.surface)
}

func TestContrastEffectHandlesReverse(t *testing.T) {
	buffer := uv.NewScreenBuffer(1, 1)
	buffer.SetCell(0, 0, &uv.Cell{Content: "x", Width: 1, Style: uv.Style{
		Fg:    ansi.RGBColor{R: 79, G: 139, B: 87},
		Bg:    ansi.RGBColor{R: 182, G: 149, B: 243},
		Attrs: uv.AttrReverse | uv.AttrBold,
	}})
	adjuster := NewHighlightContrast(nil, "#282a35", nil, true)
	highlightContrastEffect{rect: layout.Rect(0, 0, 1, 1), adjuster: adjuster}.Apply(buffer)
	cell := buffer.CellAt(0, 0)
	assert.Equal(t, uint8(uv.AttrReverse|uv.AttrBold), cell.Style.Attrs)
	assert.GreaterOrEqual(t, theme.Ratio(adjuster.rgb(cell.Style.Bg), adjuster.rgb(cell.Style.Fg)), theme.Minimum)
}
