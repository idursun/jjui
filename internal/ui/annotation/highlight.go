package annotation

import (
	"bytes"
	"image/color"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/theme"
)

type sourceHighlighter struct {
	lexers map[string]chroma.Lexer
	// cache is kept for the active source and reset by annotationRenderer when
	// that source changes, so a large source does not evict itself while rendering.
	cache           map[string]string
	formatter       chroma.Formatter
	style           *chroma.Style
	highlightMisses int
}

func newSourceHighlighter(dark bool, terminalPalettes ...map[int]string) *sourceHighlighter {
	var terminalPalette map[int]string
	if len(terminalPalettes) > 0 {
		terminalPalette = terminalPalettes[0]
	}
	return &sourceHighlighter{
		lexers:    make(map[string]chroma.Lexer),
		cache:     make(map[string]string),
		formatter: formatters.Get("terminal16m"),
		style:     syntaxStyle(dark, terminalPalette),
	}
}

func syntaxStyle(dark bool, terminalPalette map[int]string) *chroma.Style {
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
			style = style.Inherit(theme.DefaultPalette.Get("annotation", "", "text", false))
		}
		entry := chroma.StyleEntry{
			NoInherit: true,
			Colour:    syntaxColour(style.GetForeground(), terminalPalette),
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

// syntaxColour converts terminal colours to the RGB values required by Chroma.
// Prefer the terminal's actual palette; RGBA supplies standard ANSI values when
// the terminal has not reported a palette entry. NoColor must remain unset.
func syntaxColour(value color.Color, terminalPalette map[int]string) chroma.Colour {
	rgb, ok := theme.ResolveRGBFallback(value, terminalPalette)
	if !ok {
		return 0
	}
	return chroma.NewColour(rgb[0], rgb[1], rgb[2])
}

func (h *sourceHighlighter) render(path, source string) string {
	return h.highlight(path, source)
}

func (h *sourceHighlighter) renderChanged(path, source, other string, wordStyle lipgloss.Style) string {
	tokens, words := splitWordTokens(source)
	_, otherWords := splitWordTokens(other)
	changed, _ := changedWordPositions(words, otherWords)

	changedRunes := make([]bool, 0, utf8.RuneCountInString(source))
	for _, token := range tokens {
		isChanged := token.isWord && changed[token.wordIndex]
		for range token.text {
			changedRunes = append(changedRunes, isChanged)
		}
	}

	// Lex the complete line first. Lexers need the surrounding source context
	// to classify comments, strings, and other multi-token constructs correctly.
	return renderChangedRuns(h.highlight(path, source), changedRunes, wordStyle)
}

func (h *sourceHighlighter) highlight(path, source string) string {
	if source == "" {
		return ""
	}
	key := path + "\x00" + source
	if result, ok := h.cache[key]; ok {
		return result
	}
	h.highlightMisses++
	lexer := h.lexer(path, source)
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
	result := strings.TrimSuffix(out.String(), "\n")
	h.cache[key] = result
	return result
}

func (h *sourceHighlighter) resetSource() {
	clear(h.lexers)
	clear(h.cache)
}

func (h *sourceHighlighter) lexer(path, source string) chroma.Lexer {
	if lexer, ok := h.lexers[path]; ok {
		return lexer
	}
	lexer := lexers.Match(path)
	if lexer == nil {
		lexer = lexers.Analyse(source)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	h.lexers[path] = lexer
	return lexer
}

type wordToken struct {
	text      string
	wordIndex int
	isWord    bool
}

func splitWordTokens(value string) ([]wordToken, []string) {
	var tokens []wordToken
	var words []string
	for len(value) > 0 {
		first, _ := utf8.DecodeRuneInString(value)
		whitespace := unicode.IsSpace(first)
		end := 0
		for end < len(value) {
			r, size := utf8.DecodeRuneInString(value[end:])
			if unicode.IsSpace(r) != whitespace {
				break
			}
			end += size
		}
		text := value[:end]
		value = value[end:]
		token := wordToken{text: text, isWord: !whitespace}
		if token.isWord {
			token.wordIndex = len(words)
			words = append(words, text)
		}
		tokens = append(tokens, token)
	}
	return tokens, words
}

func changedWordPositions(oldWords, newWords []string) ([]bool, []bool) {
	if len(oldWords)*len(newWords) > 10000 {
		return allChanged(len(oldWords)), allChanged(len(newWords))
	}
	common := make([][]int, len(oldWords)+1)
	for i := range common {
		common[i] = make([]int, len(newWords)+1)
	}
	for i, oldWord := range slices.Backward(oldWords) {
		for j := len(newWords) - 1; j >= 0; j-- {
			if oldWord == newWords[j] {
				common[i][j] = common[i+1][j+1] + 1
			} else {
				common[i][j] = max(common[i+1][j], common[i][j+1])
			}
		}
	}
	oldChanged := allChanged(len(oldWords))
	newChanged := allChanged(len(newWords))
	for i, j := 0, 0; i < len(oldWords) && j < len(newWords); {
		if oldWords[i] == newWords[j] {
			oldChanged[i] = false
			newChanged[j] = false
			i++
			j++
		} else if common[i+1][j] >= common[i][j+1] {
			i++
		} else {
			j++
		}
	}
	return oldChanged, newChanged
}

func allChanged(length int) []bool {
	result := make([]bool, length)
	for i := range result {
		result[i] = true
	}
	return result
}

func renderChangedRuns(value string, changed []bool, style lipgloss.Style) string {
	var out strings.Builder
	background := stylePrefix(style)
	changedRun := false
	visibleIndex := 0
	for len(value) > 0 {
		if end := ansiSequenceLength(value); end > 0 {
			out.WriteString(value[:end])
			value = value[end:]
			if changedRun {
				// Chroma commonly resets SGR attributes at token boundaries. Reapply
				// the diff background after such sequences without disturbing the
				// syntax foreground.
				out.WriteString(background)
			}
			continue
		}

		_, size := utf8.DecodeRuneInString(value)
		isChanged := visibleIndex < len(changed) && changed[visibleIndex]
		if isChanged && !changedRun {
			out.WriteString(background)
			changedRun = true
		} else if !isChanged && changedRun {
			out.WriteString("\x1b[49m")
			changedRun = false
		}
		out.WriteString(value[:size])
		value = value[size:]
		visibleIndex++
	}
	if changedRun {
		out.WriteString("\x1b[49m")
	}
	return out.String()
}

func stylePrefix(style lipgloss.Style) string {
	marker := "\x00"
	rendered := style.Render(marker)
	if index := strings.IndexByte(rendered, marker[0]); index >= 0 {
		return rendered[:index]
	}
	return ""
}

func ansiSequenceLength(value string) int {
	if len(value) < 2 || value[0] != '\x1b' {
		return 0
	}
	if value[1] != '[' {
		return 2
	}
	for index := 2; index < len(value); index++ {
		if value[index] >= '@' && value[index] <= '~' {
			return index + 1
		}
	}
	return len(value)
}
