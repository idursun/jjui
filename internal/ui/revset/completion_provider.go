package revset

import (
	"strings"
	"unicode"
)

type FunctionDefinition struct {
	Name          string
	HasParameters bool
	SignatureHelp string
}

var AllFunctions = []FunctionDefinition{
	{"all", false, "all(): All commits"},
	{"mine", false, "mine(): Your own commits"},
	{"empty", false, "empty(): The empty set"},
	{"trunk", false, "trunk(): The trunk of the repository"},
	{"root", false, "root(): The root commit"},
	{"description", true, "description(pattern): Commits that have a description matching the given string pattern"},
	{"author", true, "author(pattern): Commits with the author's name or email matching the given string pattern"},
	{"author_date", true, "author_date(pattern): Commits with author dates matching the specified date pattern."},
	{"committer", true, "committer(pattern): Commits with the committer's name or email matching the given pattern"},
	{"committer_date", true, "committer_date(pattern): Commits with committer dates matching the specified date pattern"},
	{"tags", true, "tags([pattern]): All tag targets. If pattern is specified, this selects the tags whose name match the given string pattern"},
	{"files", true, "files(expression): Commits modifying paths matching the given fileset expression"},
	{"latest", true, "latest(x[, count]): Latest count commits in x"},
	{"bookmarks", true, "bookmarks([pattern]): If pattern is specified, this selects the bookmarks whose name match the given string pattern"},
	{"conflicts", false, "conflicts(): Commits with conflicts"},
	{"diff_contains", true, "diff_contains(text[, files]): Commits containing the given text in their diffs"},
	{"descendants", true, "descendants(x[, depth]): Returns the descendants of x limited to the given depth"},
	{"parents", true, "parents(x): Same as x-"},
	{"ancestors", true, "ancestors(x[, depth]): Returns the ancestors of x limited to the given depth"},
	{"connected", true, "connected(x): Same as x::x. Useful when x includes several commits"},
	{"git_head", false, "git_head(): The commit referred to by Git's HEAD"},
	{"git_refs", false, "git_refs(): All Git refs"},
	{"heads", true, "heads(x): Commits in x that are not ancestors of other commits in x"},
	{"fork_point", true, "fork_point(x): The fork point of all commits in x"},
	{"merges", true, "merges(x): Commits in x with more than one parent"},
	{"remote_bookmarks", true, "remote_bookmarks([bookmark_pattern[, [remote=]remote_pattern]]): All remote bookmarks targets across all remotes"},
	{"present", true, "present(x): Same as x, but evaluated to none() if any of the commits in x doesn't exist"},
	{"coalesce", true, "coalesce(revsets...): Commits in the first revset in the list of revsets which does not evaluate to none()"},
	{"working_copies", false, "working_copies(): All working copies"},
	{"at_operation", true, "at_operation(op, x): Evaluates to x at the specified operation"},
	{"builtin_immutable_heads", false, "builtin_immutable_heads(): Commits that the built-in mutation policy treats as immutable"},
	{"immutable", false, "immutable(): Commits that the configured mutation policy treats as immutable"},
	{"immutable_heads", false, "immutable_heads(): Heads of immutable()"},
	{"mutable", false, "mutable(): Commits that the configured mutation policy treats as mutable"},
	{"tracked_remote_bookmarks", true, "tracked_remote_bookmarks([bookmark_pattern[, [remote=]remote_pattern]])"},
	{"untracked_remote_bookmarks", true, "untracked_remote_bookmarks([bookmark_pattern[, [remote=]remote_pattern]])"},
	{"visible_heads", false, "visible_heads(): All visible heads in the repo"},
	{"reachable", true, "reachable(srcs, domain): All commits reachable from srcs within domain, traversing all parent and child edges"},
	{"roots", true, "roots(x): Commits in x that are not descendants of other commits in x"},
	{"children", true, "children(x): Same as x+"},
}

func GetFunctionByName(name string) *FunctionDefinition {
	for _, fn := range AllFunctions {
		if fn.Name == name {
			return &fn
		}
	}
	return nil
}

type CompletionProvider struct{}

func NewCompletionProvider() *CompletionProvider {
	return &CompletionProvider{}
}

func (p *CompletionProvider) GetCompletions(input string) []string {
	if input == "" {
		return nil
	}

	lastToken := getLastToken(input)
	if lastToken == "" {
		return nil
	}

	var suggestions []string
	for _, fn := range AllFunctions {
		if strings.HasPrefix(fn.Name, lastToken) {
			suggestions = append(suggestions, fn.Name)
		}
	}

	return suggestions
}

func (p *CompletionProvider) GetSignatureHelp(input string) string {
	helpFunction := extractLastFunctionName(input)
	if helpFunction == "" {
		return ""
	}

	if fn := GetFunctionByName(helpFunction); fn != nil {
		return fn.SignatureHelp
	}

	return ""
}

func (p *CompletionProvider) GetLastToken(input string) (int, string) {
	lastIndex := strings.LastIndexFunc(input, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == '|' || r == '&' || r == '~' || r == '(' || r == '.' || r == ':'
	})

	if lastIndex == -1 {
		return 0, input
	}

	return lastIndex + 1, input[lastIndex+1:]
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

func getLastToken(input string) string {
	lastIndex := strings.LastIndexFunc(input, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == '|' || r == '&' || r == '~' || r == '(' || r == '.' || r == ':'
	})

	if lastIndex == -1 {
		return input
	}

	if lastIndex+1 < len(input) {
		return input[lastIndex+1:]
	}

	return ""
}
