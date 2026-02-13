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
	{"all", false, "all(): All visible commits and ancestors of commits explicitly mentioned"},
	{"ancestors", true, "ancestors(x[, depth]): Returns the ancestors of x limited to the given depth"},
	{"at_operation", true, "at_operation(op, x): Evaluates x at the specified operation"},
	{"author", true, "author(pattern): Commits with the author's name or email matching the given string pattern"},
	{"author_date", true, "author_date(pattern): Commits with author dates matching the specified date pattern"},
	{"author_email", true, "author_email(pattern): Commits with the author's email matching the given string pattern"},
	{"author_name", true, "author_name(pattern): Commits with the author's name matching the given string pattern"},
	{"bisect", true, "bisect(x): Finds commits for which about half of the input set are descendants"},
	{"bookmarks", true, "bookmarks([pattern]): All local bookmark targets matching the given string pattern"},
	{"change_id", true, "change_id(prefix): Commits with the given change ID prefix"},
	{"children", true, "children(x[, depth]): Same as x+. With depth, returns children at the given depth"},
	{"coalesce", true, "coalesce(revsets...): Commits in the first revset which does not evaluate to none()"},
	{"commit_id", true, "commit_id(prefix): Commits with the given commit ID prefix"},
	{"committer", true, "committer(pattern): Commits with the committer's name or email matching the given pattern"},
	{"committer_date", true, "committer_date(pattern): Commits with committer dates matching the specified date pattern"},
	{"committer_email", true, "committer_email(pattern): Commits with the committer's email matching the given string pattern"},
	{"committer_name", true, "committer_name(pattern): Commits with the committer's name matching the given string pattern"},
	{"conflicts", false, "conflicts(): Commits that have files in a conflicted state"},
	{"connected", true, "connected(x): Same as x::x. Useful when x includes several commits"},
	{"descendants", true, "descendants(x[, depth]): Returns the descendants of x limited to the given depth"},
	{"description", true, "description(pattern): Commits that have a description matching the given string pattern"},
	{"diff_lines", true, "diff_lines(text[, files]): Commits containing diffs matching the given text pattern line by line"},
	{"divergent", false, "divergent(): Commits that are divergent"},
	{"empty", false, "empty(): Commits modifying no files"},
	{"exactly", true, "exactly(x, count): Evaluates x, and errors if it is not of exactly size count"},
	{"files", true, "files(expression): Commits modifying paths matching the given fileset expression"},
	{"first_ancestors", true, "first_ancestors(x[, depth]): Like ancestors but only traverses the first parent of each commit"},
	{"first_parent", true, "first_parent(x[, depth]): Like parents but only returns the first parent for merges"},
	{"fork_point", true, "fork_point(x): The fork point of all commits in x"},
	{"git_head", false, "git_head(): The commit referred to by Git's HEAD"},
	{"git_refs", false, "git_refs(): All Git refs"},
	{"heads", true, "heads(x): Commits in x that are not ancestors of other commits in x"},
	{"latest", true, "latest(x[, count]): Latest count commits in x based on committer timestamp"},
	{"merges", false, "merges(): Merge commits"},
	{"mine", false, "mine(): Commits where the author's email matches the current user"},
	{"none", false, "none(): No commits"},
	{"parents", true, "parents(x[, depth]): Same as x-. With depth, returns parents at the given depth"},
	{"present", true, "present(x): Same as x, but evaluated to none() if any of the commits in x doesn't exist"},
	{"reachable", true, "reachable(srcs, domain): All commits reachable from srcs within domain"},
	{"remote_bookmarks", true, "remote_bookmarks([name_pattern[, [remote=]remote_pattern]]): All remote bookmark targets"},
	{"remote_tags", true, "remote_tags([name_pattern[, [remote=]remote_pattern]]): All remote tag targets"},
	{"root", false, "root(): The virtual commit that is the oldest ancestor of all other commits"},
	{"roots", true, "roots(x): Commits in x that are not descendants of other commits in x"},
	{"signed", false, "signed(): Commits that are cryptographically signed"},
	{"subject", true, "subject(pattern): Commits with a subject matching the given string pattern"},
	{"tags", true, "tags([pattern]): All tag targets matching the given string pattern"},
	{"tracked_remote_bookmarks", true, "tracked_remote_bookmarks([name_pattern[, [remote=]remote_pattern]]): All tracked remote bookmark targets"},
	{"untracked_remote_bookmarks", true, "untracked_remote_bookmarks([name_pattern[, [remote=]remote_pattern]]): All untracked remote bookmark targets"},
	{"visible_heads", false, "visible_heads(): All visible heads in the repo"},
	{"working_copies", false, "working_copies(): The working copy commits across all workspaces"},
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
		items[i] = Item{Name: f.Name, Kind: KindFunction, SignatureHelp: f.SignatureHelp, HasParameters: f.HasParameters}
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
