package revset

import (
	"slices"
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
	KindRemote   = source.KindRemote

	KindArgument CompletionKind = 100
)

// CompletionItem represents a rich completion item with metadata
type CompletionItem struct {
	Name          string
	InsertText    string
	ReplaceStart  int
	SignatureHelp string
	Kind          CompletionKind
	MatchedPart   string
	RestPart      string
	HasParameters bool
}

type CompletionProvider struct {
	staticSources  []source.Source
	dynamicSources []source.Source
	sourceLoaders  map[source.CompletionSourceID][]source.Source
	functions      map[string]source.FunctionDefinition
	runner         source.Runner
	items          []source.Item
	itemsBySource  map[source.CompletionSourceID][]source.Item
}

type CompletionContext struct {
	FunctionName       string
	ArgumentIndex      int
	ArgumentName       string
	NamedArgumentValue bool
	UsedNamedArguments map[string]bool
	TokenStart         int
	Token              string
}

type functionFrame struct {
	name      string
	argIndex  int
	argStart  int
	usedNamed map[string]bool
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
		sourceLoaders: map[source.CompletionSourceID][]source.Source{
			source.CompletionRemote: {source.RemoteSource{}},
		},
		functions: source.FunctionMap(),
	}
}

func (p *CompletionProvider) Load(runner source.Runner) {
	p.runner = runner
	static := source.FetchAll(nil, p.staticSources...)
	dynamic := source.FetchAll(runner, p.dynamicSources...)
	p.items = append(static, dynamic...)
	p.itemsBySource = map[source.CompletionSourceID][]source.Item{
		source.CompletionRevset: p.items,
	}
}

func (p *CompletionProvider) GetCompletions(input string) []string {
	items := p.GetCompletionItems(input, nil)
	var suggestions []string
	for _, item := range items {
		suggestions = append(suggestions, item.InsertText)
	}
	return suggestions
}

// GetCompletionItems returns rich completion items including functions, aliases, bookmarks, tags, and history
func (p *CompletionProvider) GetCompletionItems(input string, history []string) []CompletionItem {
	p.ensureStaticLoaded()

	var items []CompletionItem

	if input == "" {
		// When input is empty, show history first for quick access,
		// followed by all available completions.
		for _, h := range history {
			items = append(items, CompletionItem{
				ReplaceStart: 0,
				Name:         h,
				InsertText:   h,
				Kind:         KindHistory,
				MatchedPart:  "",
				RestPart:     h,
			})
		}
		for _, si := range p.items {
			items = append(items, p.itemForSourceItem(si, 0, ""))
		}
		return items
	}

	ctx, inFunction := analyzeCompletionContext(input)
	if !inFunction {
		return p.matchSourceItems(p.items, ctx.TokenStart, ctx.Token, false)
	}
	return p.argumentCompletions(ctx)
}

func (p *CompletionProvider) GetSignatureHelp(input string) string {
	p.ensureStaticLoaded()

	ctx, inFunction := analyzeCompletionContext(input)
	if !inFunction || ctx.FunctionName == "" {
		return ""
	}

	for _, item := range p.items {
		if item.Name == ctx.FunctionName && item.SignatureHelp != "" {
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
		p.itemsBySource = map[source.CompletionSourceID][]source.Item{
			source.CompletionRevset: p.items,
		}
	}
}

func (p *CompletionProvider) argumentCompletions(ctx CompletionContext) []CompletionItem {
	fn, ok := p.functions[ctx.FunctionName]
	if !ok {
		return p.matchSourceItems(p.itemsBySource[source.CompletionRevset], ctx.TokenStart, ctx.Token, false)
	}

	if ctx.NamedArgumentValue {
		i := slices.IndexFunc(fn.Arguments, func(a source.FunctionArgument) bool { return a.Named && a.Name == ctx.ArgumentName })
		if i < 0 {
			return nil
		}
		arg := fn.Arguments[i]
		return p.valueCompletions(arg.CompletionSource, ctx.TokenStart, ctx.Token, true)
	}

	labelItems := p.namedArgumentCompletions(fn, ctx.TokenStart, ctx.Token, ctx.UsedNamedArguments)
	var arg source.FunctionArgument
	found := false
	index := ctx.ArgumentIndex
	for _, a := range fn.Arguments {
		if !a.Positional {
			continue
		}
		if index == 0 || a.Variadic {
			arg, found = a, true
			break
		}
		index--
	}
	if !found {
		return labelItems
	}

	valueItems := p.valueCompletions(arg.CompletionSource, ctx.TokenStart, ctx.Token, false)
	if len(valueItems) == 0 {
		return labelItems
	}
	return append(labelItems, valueItems...)
}

func (p *CompletionProvider) valueCompletions(sourceID source.CompletionSourceID, replaceStart int, token string, allowEmpty bool) []CompletionItem {
	if sourceID == source.CompletionNone {
		return nil
	}
	items, ok := p.itemsBySource[sourceID]
	if loader := p.sourceLoaders[sourceID]; !ok && loader != nil && p.runner != nil {
		if p.itemsBySource == nil {
			p.itemsBySource = make(map[source.CompletionSourceID][]source.Item)
		}
		items = source.FetchAll(p.runner, loader...)
		p.itemsBySource[sourceID] = items
	}
	return p.matchSourceItems(items, replaceStart, token, allowEmpty)
}

func (p *CompletionProvider) namedArgumentCompletions(fn source.FunctionDefinition, replaceStart int, token string, used map[string]bool) []CompletionItem {
	var items []CompletionItem
	for _, arg := range fn.Arguments {
		if !arg.Named {
			continue
		}
		if used[arg.Name] {
			continue
		}
		insertText := arg.Name + "="
		if after, ok := strings.CutPrefix(insertText, token); ok {
			items = append(items, CompletionItem{
				Name:         insertText,
				InsertText:   insertText,
				ReplaceStart: replaceStart,
				Kind:         KindArgument,
				MatchedPart:  token,
				RestPart:     after,
			})
		}
	}
	return items
}

func (p *CompletionProvider) matchSourceItems(sourceItems []source.Item, replaceStart int, token string, allowEmpty bool) []CompletionItem {
	if token == "" && !allowEmpty {
		return nil
	}

	var items []CompletionItem
	for _, si := range sourceItems {
		if after, ok := strings.CutPrefix(si.Name, token); ok {
			item := p.itemForSourceItem(si, replaceStart, token)
			item.RestPart = after
			items = append(items, item)
		}
	}
	return items
}

func (p *CompletionProvider) itemForSourceItem(si source.Item, replaceStart int, token string) CompletionItem {
	insertText := si.Name
	if si.Kind == KindFunction {
		if si.HasParameters {
			insertText += "("
		} else {
			insertText += "()"
		}
	}
	return CompletionItem{
		Name:          si.Name,
		InsertText:    insertText,
		ReplaceStart:  replaceStart,
		SignatureHelp: si.SignatureHelp,
		Kind:          si.Kind,
		MatchedPart:   token,
		RestPart:      strings.TrimPrefix(si.Name, token),
		HasParameters: si.HasParameters,
	}
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

func analyzeCompletionContext(input string) (CompletionContext, bool) {
	frames := make([]functionFrame, 0)
	var quote rune
	escaped := false

	for i, r := range input {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}

		if r == '\'' || r == '"' {
			quote = r
			continue
		}

		switch r {
		case '(':
			nameStart := i
			for nameStart > 0 {
				c := rune(input[nameStart-1])
				if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' {
					break
				}
				nameStart--
			}
			frames = append(frames, functionFrame{
				name:      input[nameStart:i],
				argIndex:  0,
				argStart:  i + len(string(r)),
				usedNamed: make(map[string]bool),
			})
		case ')':
			if len(frames) > 0 {
				frames = frames[:len(frames)-1]
			}
		case ',':
			if len(frames) > 0 {
				top := &frames[len(frames)-1]
				if before, _, ok := strings.Cut(input[top.argStart:i], "="); ok && top.usedNamed != nil {
					if name := strings.TrimSpace(before); name != "" {
						top.usedNamed[name] = true
					}
				}
				top.argIndex++
				top.argStart = i + len(string(r))
			}
		}
	}

	if len(frames) == 0 {
		start, token := lastTokenInfo(input)
		return CompletionContext{TokenStart: start, Token: token}, false
	}

	top := frames[len(frames)-1]
	argText := input[top.argStart:]
	trimmedStart := top.argStart + leadingSpaceLen(argText)
	argText = input[trimmedStart:]

	if eq := strings.IndexByte(argText, '='); eq >= 0 {
		name := strings.TrimSpace(argText[:eq])
		valueStart := trimmedStart + eq + 1
		valueText := input[valueStart:]
		valueStart += leadingSpaceLen(valueText)
		return CompletionContext{
			FunctionName:       top.name,
			ArgumentIndex:      top.argIndex,
			ArgumentName:       name,
			NamedArgumentValue: true,
			UsedNamedArguments: top.usedNamed,
			TokenStart:         valueStart,
			Token:              input[valueStart:],
		}, true
	}

	start, token := lastTokenInfo(input[trimmedStart:])
	tokenStart := trimmedStart + start
	return CompletionContext{
		FunctionName:       top.name,
		ArgumentIndex:      top.argIndex,
		UsedNamedArguments: top.usedNamed,
		TokenStart:         tokenStart,
		Token:              token,
	}, true
}

func leadingSpaceLen(input string) int {
	for i, r := range input {
		if !unicode.IsSpace(r) {
			return i
		}
	}
	return len(input)
}
