package source

import (
	"fmt"
	"strings"
)

type CompletionSourceID string

const (
	CompletionNone   CompletionSourceID = ""
	CompletionRevset CompletionSourceID = "revset"
	CompletionRemote CompletionSourceID = "remote"
)

type ArgumentKind string

const (
	ArgumentFreeform ArgumentKind = "freeform"
	ArgumentRevset   ArgumentKind = "revset"
	ArgumentNumber   ArgumentKind = "number"
)

// FunctionDefinition describes a built-in revset function.
type FunctionDefinition struct {
	Name        string
	Description string
	Arguments   []FunctionArgument
}

type FunctionArgument struct {
	Name             string
	ValueName        string
	Kind             ArgumentKind
	Optional         bool
	Variadic         bool
	Positional       bool
	Named            bool
	CompletionSource CompletionSourceID
}

var baseFunctions = []FunctionDefinition{
	revsetFunc("all", "All visible commits and ancestors of commits explicitly mentioned"),
	revsetFunc("ancestors", "Returns the ancestors of x limited to the given depth", revsetArg("x"), optionalNumberArg("depth")),
	revsetFunc("at_operation", "Evaluates x at the specified operation", freeformArg("op"), revsetArg("x")),
	revsetFunc("author", "Commits with the author's name or email matching the given string pattern", freeformArg("pattern")),
	revsetFunc("author_date", "Commits with author dates matching the specified date pattern", freeformArg("pattern")),
	revsetFunc("author_email", "Commits with the author's email matching the given string pattern", freeformArg("pattern")),
	revsetFunc("author_name", "Commits with the author's name matching the given string pattern", freeformArg("pattern")),
	revsetFunc("bisect", "Finds commits for which about half of the input set are descendants", revsetArg("x")),
	revsetFunc("bookmarks", "All local bookmark targets matching the given string pattern", optionalFreeformArg("pattern")),
	revsetFunc("change_id", "Commits with the given change ID prefix", freeformArg("prefix")),
	revsetFunc("children", "Same as x+. With depth, returns children at the given depth", revsetArg("x"), optionalNumberArg("depth")),
	revsetFunc("coalesce", "Commits in the first revset which does not evaluate to none()", variadicRevsetArg("revsets")),
	revsetFunc("commit_id", "Commits with the given commit ID prefix", freeformArg("prefix")),
	revsetFunc("committer", "Commits with the committer's name or email matching the given pattern", freeformArg("pattern")),
	revsetFunc("committer_date", "Commits with committer dates matching the specified date pattern", freeformArg("pattern")),
	revsetFunc("committer_email", "Commits with the committer's email matching the given string pattern", freeformArg("pattern")),
	revsetFunc("committer_name", "Commits with the committer's name matching the given string pattern", freeformArg("pattern")),
	revsetFunc("conflicts", "Commits that have files in a conflicted state"),
	revsetFunc("connected", "Same as x::x. Useful when x includes several commits", revsetArg("x")),
	revsetFunc("descendants", "Returns the descendants of x limited to the given depth", revsetArg("x"), optionalNumberArg("depth")),
	revsetFunc("description", "Commits that have a description matching the given string pattern", freeformArg("pattern")),
	revsetFunc("diff_lines", "Commits containing diffs matching the given text pattern line by line", freeformArg("text"), optionalFreeformArg("files")),
	revsetFunc("diff_lines_added", "Commits containing added diff lines matching the given text pattern", freeformArg("text"), optionalFreeformArg("files")),
	revsetFunc("diff_lines_removed", "Commits containing removed diff lines matching the given text pattern", freeformArg("text"), optionalFreeformArg("files")),
	revsetFunc("divergent", "Commits that are divergent"),
	revsetFunc("empty", "Commits modifying no files"),
	revsetFunc("exactly", "Evaluates x, and errors if it is not of exactly size count", revsetArg("x"), numberArg("count")),
	revsetFunc("files", "Commits modifying paths matching the given fileset expression", freeformArg("expression")),
	revsetFunc("first_ancestors", "Like ancestors but only traverses the first parent of each commit", revsetArg("x"), optionalNumberArg("depth")),
	revsetFunc("first_parent", "Like parents but only returns the first parent for merges", revsetArg("x"), optionalNumberArg("depth")),
	revsetFunc("fork_point", "The fork point of all commits in x", revsetArg("x")),
	revsetFunc("git_head", "The commit referred to by Git's HEAD"),
	revsetFunc("git_refs", "All Git refs"),
	revsetFunc("heads", "Commits in x that are not ancestors of other commits in x", revsetArg("x")),
	revsetFunc("latest", "Latest count commits in x based on committer timestamp", revsetArg("x"), optionalNumberArg("count")),
	revsetFunc("merges", "Merge commits"),
	revsetFunc("mine", "Commits where the author's email matches the current user"),
	revsetFunc("none", "No commits"),
	revsetFunc("parents", "Same as x-. With depth, returns parents at the given depth", revsetArg("x"), optionalNumberArg("depth")),
	revsetFunc("present", "Same as x, but evaluated to none() if any of the commits in x doesn't exist", revsetArg("x")),
	revsetFunc("reachable", "All commits reachable from srcs within domain", revsetArg("srcs"), revsetArg("domain")),
	remotePatternFunc("remote_bookmarks", "All remote bookmark targets"),
	remotePatternFunc("remote_tags", "All remote tag targets"),
	revsetFunc("root", "The virtual commit that is the oldest ancestor of all other commits"),
	revsetFunc("roots", "Commits in x that are not descendants of other commits in x", revsetArg("x")),
	revsetFunc("signed", "Commits that are cryptographically signed"),
	revsetFunc("subject", "Commits with a subject matching the given string pattern", freeformArg("pattern")),
	revsetFunc("tags", "All tag targets matching the given string pattern", optionalFreeformArg("pattern")),
	remotePatternFunc("tracked_remote_bookmarks", "All tracked remote bookmark targets"),
	remotePatternFunc("untracked_remote_bookmarks", "All untracked remote bookmark targets"),
	revsetFunc("visible_heads", "All visible heads in the repo"),
	revsetFunc("working_copies", "The working copy commits across all workspaces"),
}

// BaseFunctions returns a copy of the built-in function definitions.
func BaseFunctions() []FunctionDefinition {
	result := make([]FunctionDefinition, len(baseFunctions))
	copy(result, baseFunctions)
	return result
}

func FunctionMap() map[string]FunctionDefinition {
	functions := make(map[string]FunctionDefinition, len(baseFunctions))
	for _, fn := range baseFunctions {
		functions[fn.Name] = fn
	}
	return functions
}

// FunctionSource provides built-in revset functions as completion items.
type FunctionSource struct{}

func (s FunctionSource) Fetch(_ Runner) ([]Item, error) {
	items := make([]Item, len(baseFunctions))
	for i, f := range baseFunctions {
		items[i] = Item{Name: f.Name, Kind: KindFunction, SignatureHelp: f.SignatureHelp(), HasParameters: f.HasParameters()}
	}
	return items, nil
}

func (f FunctionDefinition) HasParameters() bool {
	return len(f.Arguments) > 0
}

func (f FunctionDefinition) SignatureHelp() string {
	return fmt.Sprintf("%s(%s): %s", f.Name, signatureArguments(f.Arguments), f.Description)
}

func signatureArguments(args []FunctionArgument) string {
	var b strings.Builder
	optionalDepth := 0
	for i, arg := range args {
		if arg.Optional {
			optionalDepth++
			if i > 0 {
				b.WriteString("[, ")
			} else {
				b.WriteString("[")
			}
		} else if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.SignatureName())
	}
	for range optionalDepth {
		b.WriteString("]")
	}
	return b.String()
}

func (a FunctionArgument) SignatureName() string {
	name := a.ValueName
	if name == "" {
		name = a.Name
	}
	if a.Variadic {
		name += "..."
	}
	if a.Named {
		return "[" + a.Name + "=]" + name
	}
	return name
}

func revsetFunc(name string, description string, args ...FunctionArgument) FunctionDefinition {
	return FunctionDefinition{Name: name, Description: description, Arguments: args}
}

func remotePatternFunc(name string, description string) FunctionDefinition {
	return revsetFunc(name, description,
		optionalFreeformArg("name_pattern"),
		FunctionArgument{
			Name:             "remote",
			ValueName:        "remote_pattern",
			Kind:             ArgumentFreeform,
			Optional:         true,
			Positional:       true,
			Named:            true,
			CompletionSource: CompletionRemote,
		},
	)
}

func freeformArg(name string) FunctionArgument {
	return FunctionArgument{Name: name, Kind: ArgumentFreeform, Positional: true}
}

func optionalFreeformArg(name string) FunctionArgument {
	arg := freeformArg(name)
	arg.Optional = true
	return arg
}

func revsetArg(name string) FunctionArgument {
	return FunctionArgument{Name: name, Kind: ArgumentRevset, Positional: true, CompletionSource: CompletionRevset}
}

func variadicRevsetArg(name string) FunctionArgument {
	arg := revsetArg(name)
	arg.Variadic = true
	return arg
}

func numberArg(name string) FunctionArgument {
	return FunctionArgument{Name: name, Kind: ArgumentNumber, Positional: true}
}

func optionalNumberArg(name string) FunctionArgument {
	arg := numberArg(name)
	arg.Optional = true
	return arg
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

		openParen := strings.Index(alias, "(")
		closeParen := strings.LastIndex(alias, ")")
		if openParen >= 0 && closeParen == len(alias)-1 && openParen < closeParen {
			name = alias[:openParen]
			hasParameters = strings.TrimSpace(alias[openParen+1:closeParen]) != ""
		}

		items = append(items, Item{
			Name:          name,
			Kind:          KindAlias,
			SignatureHelp: signatureHelp,
			HasParameters: hasParameters,
		})
	}
	return items, nil
}
