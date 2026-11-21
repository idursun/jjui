package syntax

import (
	"bytes"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/theme"
)

// Highlighter colours source code with chroma, picking the lexer from the
// file name and falling back to guessing from the source.
type Highlighter struct {
	lexers    map[string]chroma.Lexer
	formatter chroma.Formatter
	style     *chroma.Style
}

func New(dark bool, scope string, terminalPalette map[int]string) *Highlighter {
	return &Highlighter{
		lexers:    make(map[string]chroma.Lexer),
		formatter: formatters.Get("terminal16m"),
		style:     Style(dark, scope, terminalPalette),
	}
}

// Style maps the theme's syntax colours onto chroma tokens. Plain text also
// inherits the text style of scope, the view the source is drawn in.
func Style(dark bool, scope string, terminalPalette map[int]string) *chroma.Style {
	// Custom themes replace the embedded theme, so supply syntax defaults here
	// too, while allowing explicitly configured attributes to take precedence.
	defaults := theme.NewPalette()
	resolvedTheme, err := config.LoadEmbeddedTheme("default", dark)
	if err == nil {
		defaults.Update(resolvedTheme.Colors)
	}
	roles := map[chroma.TokenType][]string{
		chroma.Text:          {"text"},
		chroma.Keyword:       {"keyword"},
		chroma.LiteralString: {"string"},
		chroma.LiteralNumber: {"number"},
		chroma.Comment:       {"comment"},
		chroma.NameFunction:  {"function"},
		chroma.NameClass:     {"class", "type"},
		chroma.KeywordType:   {"type"},
	}
	entries := chroma.StyleEntries{}
	for token := range chroma.StandardTypes {
		fallbacks, ok := roles[token]
		if !ok {
			fallbacks, ok = roles[token.SubCategory()]
		}
		if !ok {
			fallbacks, ok = roles[token.Category()]
		}
		if !ok {
			fallbacks = []string{"text"}
		}
		style := lipgloss.NewStyle()
		// Resolve explicit syntax selectors before broad palette defaults, so an
		// absent class style can inherit type even when syntax has a base style.
		for _, role := range fallbacks {
			style = style.Inherit(theme.DefaultPalette.Get("", "", "syntax "+role, false))
		}
		role := fallbacks[len(fallbacks)-1]
		style = style.Inherit(theme.DefaultPalette.Get("", "syntax", role, false))
		for _, role := range fallbacks {
			style = style.Inherit(defaults.Get("", "syntax", role, false))
		}
		if role == "text" {
			style = style.Inherit(theme.DefaultPalette.Get(scope, "", "text", false))
		}
		// Chroma needs RGB; NoColor and unreported palette entries stay unset.
		var colour chroma.Colour
		if rgb, ok := theme.ResolveRGBFallback(style.GetForeground(), terminalPalette); ok {
			colour = chroma.NewColour(rgb[0], rgb[1], rgb[2])
		}
		entry := chroma.StyleEntry{
			NoInherit: true,
			Colour:    colour,
			Bold:      chroma.No, Italic: chroma.No, Underline: chroma.No,
		}
		if style.GetBold() {
			entry.Bold = chroma.Yes
		}
		if style.GetItalic() {
			entry.Italic = chroma.Yes
		}
		if style.GetUnderline() {
			entry.Underline = chroma.Yes
		}
		// Backgrounds belong to the diff and selection layers.
		entries[token] = entry.String()
	}
	return chroma.MustNewStyle("jjui", entries)
}

// Highlight returns source with ANSI colours, or source unchanged when it
// cannot be lexed.
func (h *Highlighter) Highlight(path, source string) string {
	if source == "" {
		return ""
	}
	lexer, ok := h.lexers[path]
	if !ok {
		lexer = lexers.Match(path)
		if lexer == nil {
			lexer = lexers.Analyse(source)
		}
		if lexer == nil {
			lexer = lexers.Fallback
		}
		lexer = chroma.Coalesce(lexer)
		h.lexers[path] = lexer
	}
	if lexer == nil || h.formatter == nil {
		return source
	}
	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return source
	}
	var out bytes.Buffer
	if err := h.formatter.Format(&out, h.style, iterator); err != nil {
		return source
	}
	return strings.TrimSuffix(out.String(), "\n")
}

// Reset forgets the lexers picked so far.
func (h *Highlighter) Reset() {
	clear(h.lexers)
}
