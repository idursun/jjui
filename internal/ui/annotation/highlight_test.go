package annotation

import (
	"strconv"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangedWordBackgroundPreservesSyntaxForeground(t *testing.T) {
	highlighter := newSourceHighlighter(true)
	wordBackground := lipgloss.NewStyle().Background(lipgloss.Color("2"))
	rendered := highlighter.renderChanged("example.go", `return "new"`, `return "old"`, wordBackground)

	buffer := uv.NewScreenBuffer(12, 1)
	uv.NewStyledString(rendered).Draw(buffer, layout.Rect(0, 0, 12, 1))
	changedCell := buffer.CellAt(8, 0)
	require.NotNil(t, changedCell)
	require.NotNil(t, changedCell.Style.Fg)
	require.NotNil(t, changedCell.Style.Bg)
	changedWordBackground := changedCell.Style.Bg

	render.HighlightEffect{
		Rect:  layout.Rect(0, 0, 12, 1),
		Style: lipgloss.NewStyle().Background(lipgloss.Color("52")),
	}.Apply(buffer)

	plainCell := buffer.CellAt(0, 0)
	changedCell = buffer.CellAt(8, 0)
	assert.NotNil(t, plainCell.Style.Bg)
	assert.NotNil(t, changedCell.Style.Fg)
	assert.Equal(t, changedWordBackground, changedCell.Style.Bg)
}

func TestChangedWordHighlightUsesCompleteLineContext(t *testing.T) {
	highlighter := newSourceHighlighter(true)
	wordBackground := lipgloss.NewStyle().Background(lipgloss.Color("2"))
	rendered := highlighter.renderChanged("example.go", `// changed`, `// original`, wordBackground)

	buffer := uv.NewScreenBuffer(10, 1)
	uv.NewStyledString(rendered).Draw(buffer, layout.Rect(0, 0, 10, 1))
	commentCell := buffer.CellAt(1, 0)
	changedCell := buffer.CellAt(3, 0)
	require.NotNil(t, commentCell)
	require.NotNil(t, changedCell)
	require.NotNil(t, commentCell.Style.Fg)
	require.NotNil(t, changedCell.Style.Fg)
	assert.Equal(t, commentCell.Style.Fg, changedCell.Style.Fg)
}

func TestHighlightCacheRetainsTheCurrentSource(t *testing.T) {
	highlighter := newSourceHighlighter(true)
	for index := range 320 {
		highlighter.highlight("example.go", "value "+strconv.Itoa(index))
	}

	assert.Len(t, highlighter.cache, 320)
	assert.Equal(t, 320, highlighter.highlightMisses)

	highlighter.resetSource()
	assert.Empty(t, highlighter.cache)
}

func TestSyntaxStyleUsesThemeAndTerminalPalette(t *testing.T) {
	previous := theme.DefaultPalette
	t.Cleanup(func() { theme.DefaultPalette = previous })
	theme.DefaultPalette = theme.NewPalette()
	bold := false
	theme.DefaultPalette.Update(map[string]config.Color{
		"syntax keyword": {Fg: "magenta", Bold: &bold, Bg: "red"},
		"syntax text":    {Fg: "#eeeeee"},
		"syntax string":  {Fg: "#123456"},
	})
	style := syntaxStyle(true, map[int]string{5: "#abcdef"})
	assert.Equal(t, chroma.MustParseColour("#abcdef"), style.Get(chroma.KeywordDeclaration).Colour)
	assert.Equal(t, chroma.No, style.Get(chroma.KeywordDeclaration).Bold)
	assert.False(t, style.Get(chroma.KeywordDeclaration).Background.IsSet())
	assert.Equal(t, chroma.MustParseColour("#123456"), style.Get(chroma.LiteralStringDouble).Colour)
	assert.Equal(t, chroma.MustParseColour("#eeeeee"), style.Get(chroma.Name).Colour)
	assert.Equal(t, chroma.Yes, style.Get(chroma.CommentSingle).Italic)
	assert.NotEqual(t, style.Get(chroma.Name).Colour, style.Get(chroma.CommentSingle).Colour)
}

func TestSyntaxClassFallsBackToType(t *testing.T) {
	previous := theme.DefaultPalette
	t.Cleanup(func() { theme.DefaultPalette = previous })
	theme.DefaultPalette = theme.NewPalette()
	bold := true
	for _, tc := range []struct {
		name       string
		class      config.Color
		wantColour string
	}{
		{name: "absent", wantColour: "#123456"},
		{name: "explicit", class: config.Color{Fg: "#abcdef"}, wantColour: "#abcdef"},
		{name: "partial", class: config.Color{Bold: &bold}, wantColour: "#123456"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			colors := map[string]config.Color{
				"syntax":      {Fg: "#eeeeee"},
				"syntax type": {Fg: "#123456"},
			}
			if tc.name != "absent" {
				colors["syntax class"] = tc.class
			}
			theme.DefaultPalette.Update(colors)
			style := syntaxStyle(true, nil)
			assert.Equal(t, chroma.MustParseColour(tc.wantColour), style.Get(chroma.NameClass).Colour)
			assert.Equal(t, chroma.MustParseColour("#123456"), style.Get(chroma.KeywordType).Colour)
			if tc.name == "partial" {
				assert.Equal(t, chroma.Yes, style.Get(chroma.NameClass).Bold)
			}
		})
	}
}
