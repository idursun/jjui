package syntax

import (
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

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
	style := Style(true, "annotation", map[int]string{5: "#abcdef"})
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
			style := Style(true, "annotation", nil)
			assert.Equal(t, chroma.MustParseColour(tc.wantColour), style.Get(chroma.NameClass).Colour)
			assert.Equal(t, chroma.MustParseColour("#123456"), style.Get(chroma.KeywordType).Colour)
			if tc.name == "partial" {
				assert.Equal(t, chroma.Yes, style.Get(chroma.NameClass).Bold)
			}
		})
	}
}

func TestHighlightPicksTheLexerFromThePath(t *testing.T) {
	h := New(true, "preview", nil)
	assert.Contains(t, h.Highlight("main.go", "package main\n"), "\x1b[")
	assert.Empty(t, h.Highlight("main.go", ""))
}
