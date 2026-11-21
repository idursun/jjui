package annotation

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/ui/syntax"
)

type sourceHighlighter struct {
	syntax *syntax.Highlighter
	// cache is kept for the active source and reset by annotationRenderer when
	// that source changes, so a large source does not evict itself while rendering.
	cache           map[string]string
	highlightMisses int
}

func newSourceHighlighter(dark bool, terminalPalettes ...map[int]string) *sourceHighlighter {
	var terminalPalette map[int]string
	if len(terminalPalettes) > 0 {
		terminalPalette = terminalPalettes[0]
	}
	return &sourceHighlighter{
		syntax: syntax.New(dark, "annotation", terminalPalette),
		cache:  make(map[string]string),
	}
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
	result := h.syntax.Highlight(path, source)
	h.cache[key] = result
	return result
}

func (h *sourceHighlighter) resetSource() {
	h.syntax.Reset()
	clear(h.cache)
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
	background := style.Render("\x00")
	if index := strings.IndexByte(background, 0); index >= 0 {
		background = background[:index]
	} else {
		background = ""
	}
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
