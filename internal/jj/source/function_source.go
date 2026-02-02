package source

import (
	"fmt"
	"strings"
)

// FunctionDefinition describes a built-in revset function.
type FunctionDefinition struct {
	Name          string
	HasParameters bool
	SignatureHelp string
}

var baseFunctions = []FunctionDefinition{
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
	{"tracked_remote_bookmarks", true, "tracked_remote_bookmarks([bookmark_pattern[, [remote=]remote_pattern]])"},
	{"untracked_remote_bookmarks", true, "untracked_remote_bookmarks([bookmark_pattern[, [remote=]remote_pattern]])"},
	{"visible_heads", false, "visible_heads(): All visible heads in the repo"},
	{"reachable", true, "reachable(srcs, domain): All commits reachable from srcs within domain, traversing all parent and child edges"},
	{"roots", true, "roots(x): Commits in x that are not descendants of other commits in x"},
	{"children", true, "children(x): Same as x+"},
}

// BaseFunctions returns a copy of the built-in function definitions.
func BaseFunctions() []FunctionDefinition {
	result := make([]FunctionDefinition, len(baseFunctions))
	copy(result, baseFunctions)
	return result
}

// FunctionSource provides built-in revset functions as completion items.
type FunctionSource struct{}

func (s FunctionSource) Fetch(_ Runner) ([]Item, error) {
	items := make([]Item, len(baseFunctions))
	for i, f := range baseFunctions {
		items[i] = Item{Name: f.Name, Kind: KindFunction, SignatureHelp: f.SignatureHelp}
	}
	return items, nil
}

// AliasSource converts revset aliases into completion items.
type AliasSource struct {
	Aliases map[string]string
}

func (s AliasSource) Fetch(_ Runner) ([]Item, error) {
	var items []Item
	for alias, expansion := range s.Aliases {
		name := alias
		hasParameters := false
		signatureHelp := fmt.Sprintf("%s: %s", alias, expansion)

		if strings.Index(alias, "(") < strings.LastIndex(alias, ")") {
			hasParameters = true
			name = alias[:strings.Index(alias, "(")]
		} else if strings.HasSuffix(alias, "()") {
			hasParameters = false
			name = alias[:len(alias)-2]
		}
		_ = hasParameters

		items = append(items, Item{Name: name, Kind: KindAlias, SignatureHelp: signatureHelp})
	}
	return items, nil
}
