package revset

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
)

var allowedFunctions = []string{
	"all()",
	"mine()",
	"empty()",
	"trunk()",
	"root()",
	"description(",
	"author(",
	"author_date(",
	"committer(",
	"committer_date(",
	"tags(",
	"files(",
	"latest(",
	"bookmarks(",
	"conflicts()",
	"diff_contains(",
	"descendants(",
	"parents(",
	"ancestors(",
	"connected(",
	"git_head()",
	"git_refs()",
	"heads(",
	"fork_point(",
	"merges(",
	"remote_bookmarks(",
	"present(",
	"coalesce(",
	"working_copies()",
	"at_operation(",
	"builtin_immutable_heads()",
	"immutable()",
	"immutable_heads()",
	"mutable()",
	"tracked_remote_bookmarks(",
	"untracked_remote_bookmarks(",
	"visible_heads()",
	"reachable(",
	"roots(",
	"children(",
}

var functionSignatureHelp = map[string]string{
	"parents":                    "parents(x): Same as x-",
	"children":                   "children(x): Same as x+",
	"ancestors":                  "ancestors(x[, depth]): Returns the ancestors of x limited to the given depth",
	"descendants":                "descendants(x[, depth]): Returns the descendants of x limited to the given depth",
	"reachable":                  "reachable(srcs, domain): All commits reachable from srcs within domain, traversing all parent and child edges",
	"connected":                  "connected(x): Same as x::x. Useful when x includes several commits",
	"bookmarks":                  "bookmarks([pattern]): If pattern is specified, this selects the bookmarks whose name match the given string pattern",
	"remote_bookmarks":           "remote_bookmarks([bookmark_pattern[, [remote=]remote_pattern]]): All remote bookmarks targets across all remotes",
	"tracked_remote_bookmarks":   "tracked_remote_bookmarks([bookmark_pattern[, [remote=]remote_pattern]])",
	"untracked_remote_bookmarks": "untracked_remote_bookmarks([bookmark_pattern[, [remote=]remote_pattern]])",
	"tags":                       "tags([pattern]): All tag targets. If pattern is specified, this selects the tags whose name match the given string pattern",
	"heads":                      "heads(x): Commits in x that are not ancestors of other commits in x",
	"roots":                      "roots(x): Commits in x that are not descendants of other commits in x",
	"latest":                     "latest(x[, count]): Latest count commits in x",
	"fork_point":                 "fork_point(x): The fork point of all commits in x",
	"description":                "description(pattern): Commits that have a description matching the given string pattern",
	"author":                     "author(pattern): Commits with the author's name or email matching the given string pattern",
	"committer":                  "committer(pattern): Commits with the committer's name or email matching the given pattern",
	"author_date":                "author_date(pattern): Commits with author dates matching the specified date pattern.",
	"committer_date":             "committer_date(pattern): Commits with committer dates matching the specified date pattern",
	"files":                      "files(expression): Commits modifying paths matching the given fileset expression",
	"diff_contains":              "diff_contains(text[, files]): Commits containing the given text in their diffs",
	"present":                    "present(x): Same as x, but evaluated to none() if any of the commits in x doesn't exist",
	"coalesce":                   "coalesce(revsets...): Commits in the first revset in the list of revsets which does not evaluate to none()",
	"at_operation":               "at_operation(op, x): Evaluates to x at the specified operation",
}

type EditRevSetMsg struct{}

type Model struct {
	Editing       bool
	Value         string
	defaultRevSet string
	signatureHelp string
	textInput     textinput.Model
	help          help.Model
	keymap        keymap
}

var (
	promptStyle = common.DefaultPalette.CommitShortStyle
	cursorStyle = common.DefaultPalette.Empty
)

type keymap struct{}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "complete")),
		key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "next")),
		key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "prev")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "quit")),
	}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

func New(defaultRevSet string) Model {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Prompt = "revset: "
	ti.PromptStyle = promptStyle
	ti.Cursor.Style = cursorStyle
	ti.Focus()
	ti.ShowSuggestions = true
	ti.SetValue(defaultRevSet)

	h := help.New()
	h.Styles.ShortKey = common.DefaultPalette.CommitShortStyle
	h.Styles.ShortDesc = common.DefaultPalette.CommitIdRestStyle
	return Model{
		Editing:       false,
		Value:         defaultRevSet,
		defaultRevSet: defaultRevSet,
		help:          h,
		keymap:        keymap{},
		textInput:     ti,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Editing = false
			return m, nil
		case tea.KeyEnter:
			m.Editing = false
			m.Value = m.textInput.Value()
			if m.Value == "" {
				m.Value = m.defaultRevSet
			}
			return m, tea.Batch(common.Close, common.UpdateRevSet(m.Value))
		}
	case EditRevSetMsg:
		m.Editing = true
		m.signatureHelp = ""
		m.textInput.Focus()
		m.textInput.SetValue("")
		return m, textinput.Blink
	}

	value := m.textInput.Value()
	var suggestions []string
	lastIndex := strings.LastIndexFunc(strings.Trim(value, "() "), func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == '|' || r == '&' || r == '~' || r == '('
	})

	if lastIndex == -1 && value == "" {
		suggestions = []string{"@ | mine()"}
	} else {
		lastFunctionName := value[lastIndex+1:]
		m.signatureHelp = ""
		helpFunction := strings.Trim(lastFunctionName, "() ")
		if _, ok := functionSignatureHelp[helpFunction]; ok {
			m.signatureHelp = functionSignatureHelp[helpFunction]
		}
		if !strings.HasSuffix(value, ")") && lastFunctionName != "" {
			for _, f := range allowedFunctions {
				if strings.HasPrefix(f, lastFunctionName) {
					rest := strings.TrimPrefix(f, lastFunctionName)
					suggestions = append(suggestions, value+rest)
				}
			}
		}
	}
	m.textInput.SetSuggestions(suggestions)

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.Editing {
		if m.signatureHelp != "" {
			return lipgloss.JoinVertical(0, m.textInput.View(), m.signatureHelp)
		}
		return lipgloss.JoinVertical(0, m.textInput.View(), m.help.View(m.keymap))
	}

	revset := "(default)"
	if m.Value != "" {
		revset = m.Value
	}

	return fmt.Sprintf("%s: %s", promptStyle.Render("revset"), cursorStyle.Render(revset))
}
