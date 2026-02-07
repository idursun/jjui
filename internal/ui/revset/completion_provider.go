package revset

import (
	"strings"
	"unicode"

	"github.com/idursun/jjui/internal/jj/source"
)

// CompletionKind represents the type of completion item
type CompletionKind = source.Kind

const (
	KindFunction = source.KindFunction
	KindAlias    = source.KindAlias
	KindHistory  = source.KindHistory
	KindBookmark = source.KindBookmark
	KindTag      = source.KindTag
)

// CompletionItem represents a rich completion item with metadata
type CompletionItem struct {
	Name          string
	SignatureHelp string
	Kind          CompletionKind
	MatchedPart   string
	RestPart      string
}

type CompletionProvider struct {
	staticSources  []source.Source
	dynamicSources []source.Source
	items          []source.Item
}

func NewCompletionProvider(aliases map[string]string) *CompletionProvider {
	return &CompletionProvider{
		staticSources: []source.Source{
			source.FunctionSource{},
			source.AliasSource{Aliases: aliases},
		},
		dynamicSources: []source.Source{
			source.BookmarkSource{},
			source.TagSource{},
		},
	}
}

func (p *CompletionProvider) Load(runner source.Runner) {
	static := source.FetchAll(nil, p.staticSources...)
	dynamic := source.FetchAll(runner, p.dynamicSources...)
	p.items = append(static, dynamic...)
}

func (p *CompletionProvider) GetCompletions(input string) []string {
	p.ensureStaticLoaded()

	var suggestions []string
	if input == "" {
		for _, item := range p.items {
			if item.Kind == KindAlias {
				suggestions = append(suggestions, item.Name)
			}
		}
		return suggestions
	}

	_, lastToken := p.GetLastToken(input)
	if lastToken == "" {
		return nil
	}

	for _, item := range p.items {
		if item.Kind == KindFunction || item.Kind == KindAlias {
			if strings.HasPrefix(item.Name, lastToken) {
				suggestions = append(suggestions, item.Name)
			}
		}
	}

	return suggestions
}

// GetCompletionItems returns rich completion items including functions, aliases, bookmarks, tags, and history
func (p *CompletionProvider) GetCompletionItems(input string, history []string) []CompletionItem {
	p.ensureStaticLoaded()

	var items []CompletionItem

	if input == "" {
		// When input is empty, show history for quick access
		for _, h := range history {
			items = append(items, CompletionItem{
				Name:        h,
				Kind:        KindHistory,
				MatchedPart: "",
				RestPart:    h,
			})
		}
		if len(items) > 0 {
			return items
		}
		// No history: fall through to show all available completions
		for _, si := range p.items {
			items = append(items, CompletionItem{
				Name:          si.Name,
				SignatureHelp: si.SignatureHelp,
				Kind:          si.Kind,
				MatchedPart:   "",
				RestPart:      si.Name,
			})
		}
		return items
	}

	_, lastToken := p.GetLastToken(input)
	if lastToken == "" {
		return nil
	}

	for _, si := range p.items {
		if strings.HasPrefix(si.Name, lastToken) {
			items = append(items, CompletionItem{
				Name:          si.Name,
				SignatureHelp: si.SignatureHelp,
				Kind:          si.Kind,
				MatchedPart:   lastToken,
				RestPart:      strings.TrimPrefix(si.Name, lastToken),
			})
		}
	}

	return items
}

func (p *CompletionProvider) GetSignatureHelp(input string) string {
	p.ensureStaticLoaded()

	helpFunction := extractLastFunctionName(input)
	if helpFunction == "" {
		return ""
	}

	for _, item := range p.items {
		if item.Name == helpFunction && item.SignatureHelp != "" {
			return item.SignatureHelp
		}
	}

	return ""
}

func (p *CompletionProvider) GetLastToken(input string) (int, string) {
	return lastTokenInfo(input)
}

// ensureStaticLoaded loads static sources if items haven't been loaded yet.
func (p *CompletionProvider) ensureStaticLoaded() {
	if p.items == nil {
		p.items = source.FetchAll(nil, p.staticSources...)
	}
}

func extractLastFunctionName(input string) string {
	lastOpenParen := strings.LastIndex(input, "(")
	if lastOpenParen == -1 {
		return ""
	}

	parenCount := 1
	for i := lastOpenParen + 1; i < len(input); i++ {
		if input[i] == '(' {
			parenCount++
		} else if input[i] == ')' {
			parenCount--
		}

		if parenCount == 0 && i+1 < len(input) {
			for j := i + 1; j < len(input); j++ {
				ch := input[j]
				if ch == '|' || ch == '&' || ch == ',' || !unicode.IsSpace(rune(ch)) {
					return ""
				}

				if !unicode.IsSpace(rune(ch)) {
					break
				}
			}
			break
		}
	}

	startIndex := lastOpenParen
	for startIndex > 0 {
		startIndex--
		if !isValidFunctionNameChar(rune(input[startIndex])) {
			startIndex++ // Move back to the valid character
			break
		}
	}

	if startIndex <= lastOpenParen {
		funcName := input[startIndex:lastOpenParen]
		return funcName
	}

	return ""
}

func isValidFunctionNameChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func lastTokenInfo(input string) (int, string) {
	lastIndex := strings.LastIndexFunc(input, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == '|' || r == '&' || r == '~' || r == '(' || r == '.' || r == ':'
	})

	if lastIndex == -1 {
		return 0, input
	}

	if lastIndex+1 < len(input) {
		return lastIndex + 1, input[lastIndex+1:]
	}

	return len(input), ""
}
