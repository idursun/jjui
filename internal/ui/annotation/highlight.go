package annotation

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

type sourceHighlighter struct {
	lexers    map[string]chroma.Lexer
	cache     map[string]string
	formatter chroma.Formatter
	style     *chroma.Style
}

func newSourceHighlighter(dark bool) *sourceHighlighter {
	styleName := "github"
	if dark {
		styleName = "github-dark"
	}
	style := styles.Get(styleName)
	if style == nil {
		style = styles.Fallback
	}
	return &sourceHighlighter{
		lexers:    make(map[string]chroma.Lexer),
		cache:     make(map[string]string),
		formatter: formatters.Get("terminal16m"),
		style:     style,
	}
}

func (h *sourceHighlighter) render(path, source string) string {
	return h.highlight(path, source)
}

func (h *sourceHighlighter) renderChanged(path, source, other string, wordStyle lipgloss.Style) string {
	tokens, words := splitWordTokens(source)
	_, otherWords := splitWordTokens(other)
	changed, _ := changedWordPositions(words, otherWords)

	var out strings.Builder
	for _, token := range tokens {
		switch {
		case token.isWord && changed[token.wordIndex]:
			out.WriteString(renderVisibleRuns(h.highlight(path, token.text), wordStyle))
		case token.isWord:
			out.WriteString(h.highlight(path, token.text))
		default:
			out.WriteString(token.text)
		}
	}
	return out.String()
}

func (h *sourceHighlighter) highlight(path, source string) string {
	if source == "" {
		return ""
	}
	key := path + "\x00" + source
	if result, ok := h.cache[key]; ok {
		return result
	}
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
	for i := len(oldWords) - 1; i >= 0; i-- {
		for j := len(newWords) - 1; j >= 0; j-- {
			if oldWords[i] == newWords[j] {
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

func renderVisibleRuns(value string, style lipgloss.Style) string {
	var out strings.Builder
	for len(value) > 0 {
		if strings.HasPrefix(value, "\x1b[") {
			end := 2
			for end < len(value) && (value[end] < '@' || value[end] > '~') {
				end++
			}
			if end < len(value) {
				end++
			}
			out.WriteString(value[:end])
			value = value[end:]
			continue
		}
		index := strings.Index(value, "\x1b[")
		if index < 0 {
			out.WriteString(style.Render(value))
			break
		}
		out.WriteString(style.Render(value[:index]))
		value = value[index:]
	}
	return out.String()
}
