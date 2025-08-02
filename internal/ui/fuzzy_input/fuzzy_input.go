package fuzzy_input

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/sahilm/fuzzy"
)

type historyMode int

const (
	historyOff historyMode = iota
	historyFuzzy
	historyExact
)

const ctrl_r = "ctrl+r"

type model struct {
	suggestions []string
	input       *textinput.Model
	cursor      int
	max         int
	matches     fuzzy.Matches
	styles      fuzzy_search.Styles
	historyMode historyMode
}

type initMsg struct{}

func newCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func (fzf *model) Init() tea.Cmd {
	return newCmd(initMsg{})
}

func (fzf *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initMsg:
		fzf.search("")
	case fuzzy_search.SearchMsg:
		if cmd := fzf.handleKey(msg.Pressed); cmd != nil {
			return fzf, cmd
		} else {
			fzf.search(msg.Input)
		}
	case tea.KeyMsg:
		return fzf, fzf.handleKey(msg)
	}
	return fzf, nil
}

func (fzf *model) handleKey(msg tea.KeyMsg) tea.Cmd {
	km := config.Current.GetKeyMap()
	skipSearch := func() tea.Msg { return nil }
	switch {
	case ctrl_r == msg.String():
		switch fzf.historyMode {
		case historyOff:
			fzf.historyMode = historyFuzzy
			return nil
		case historyFuzzy:
			fzf.historyMode = historyExact
			return nil
		case historyExact:
			fzf.historyMode = historyOff
			return nil
		}
	case key.Matches(msg, fzf.input.KeyMap.AcceptSuggestion) && fzf.historyMode != historyOff && len(fzf.matches) > 0:
		suggestion := fuzzy_search.SelectedMatch(fzf)
		fzf.input.SetValue(suggestion)
		fzf.input.CursorEnd()
		return skipSearch
	case key.Matches(msg, km.Up, km.Preview.ScrollUp, fzf.input.KeyMap.PrevSuggestion):
		fzf.moveCursor(1)
		return skipSearch
	case key.Matches(msg, km.Down, km.Preview.ScrollDown, fzf.input.KeyMap.NextSuggestion):
		fzf.moveCursor(-1)
		return skipSearch
	case key.Matches(msg,
		// movements do not cause search
		fzf.input.KeyMap.CharacterForward,
		fzf.input.KeyMap.CharacterBackward,
		fzf.input.KeyMap.WordForward,
		fzf.input.KeyMap.WordBackward,
		fzf.input.KeyMap.LineStart,
		fzf.input.KeyMap.LineEnd,
	):
		return skipSearch
	}
	return nil
}

func (fzf *model) moveCursor(inc int) {
	l := len(fzf.matches)
	n := fzf.cursor + inc
	if n < 0 {
		n = l - 1
	}
	if n >= l {
		n = 0
	}
	fzf.cursor = n
}

func (fzf *model) Styles() fuzzy_search.Styles {
	return fzf.styles
}

func (fzf *model) Max() int {
	return fzf.max
}

func (fzf *model) Matches() fuzzy.Matches {
	return fzf.matches
}

func (fzf *model) SelectedMatch() int {
	return fzf.cursor
}

func (fzf *model) Len() int {
	return len(fzf.suggestions)
}

func (fzf *model) String(i int) string {
	if len(fzf.suggestions) == 0 {
		return ""
	}
	return fzf.suggestions[i]
}

func (fzf *model) search(input string) {
	input = strings.TrimSpace(input)
	fzf.cursor = 0
	fzf.matches = fuzzy.Matches{}
	if len(input) == 0 {
		return
	}
	if fzf.historyMode == historyFuzzy {
		fzf.matches = fuzzy.FindFrom(input, fzf)
	} else if fzf.historyMode == historyExact {
		fzf.matches = fzf.searchSubstr(input)
	}
}

func (fzf *model) searchSubstr(input string) fuzzy.Matches {
	parts := strings.Fields(input)
	matches := fuzzy.Matches{}
	for i := range fzf.Len() {
		item := strings.Fields(fzf.String(i))
		deleted := slices.DeleteFunc(parts, func(p string) bool {
			return slices.ContainsFunc(item, func(s string) bool {
				return strings.Contains(s, p)
			})
		})
		if len(deleted) == 0 {
			matches = append(matches, fuzzy.Match{
				Index: i,
			})
		}
	}
	return matches
}

func (fzf *model) View() string {
	matches := len(fzf.matches)
	if matches == 0 {
		return ""
	}
	view := fuzzy_search.View(fzf)
	title := fmt.Sprintf(
		"  %s of %s elements in history ",
		strconv.Itoa(matches),
		strconv.Itoa(fzf.Len()),
	)
	title = fzf.styles.SelectedMatch.Render(title)
	return lipgloss.JoinVertical(0, title, view)
}

func (fzf *model) ShortHelp() []key.Binding {
	short_help := []key.Binding{}
	bind := func(keys string, help string) key.Binding {
		return key.NewBinding(key.WithKeys(keys), key.WithHelp(keys, help))
	}

	switch fzf.historyMode {
	case historyOff:
		short_help = append(short_help, bind("ctrl+r", "history off"))
	case historyFuzzy:
		short_help = append(short_help, bind("ctrl+r", "fuzzy history"))
	case historyExact:
		short_help = append(short_help, bind("ctrl+r", "exact history"))
	}
	return short_help
}

func (fzf *model) FullHelp() [][]key.Binding {
	return [][]key.Binding{fzf.ShortHelp()}
}

type editStatus func() (help.KeyMap, string)

func (fzf *model) editStatus() (help.KeyMap, string) {
	return fzf, ""
}

func NewModel(input *textinput.Model, suggestions []string) (fuzzy_search.Model, editStatus) {
	input.ShowSuggestions = false
	input.SetSuggestions([]string{})
	fzf := &model{
		input:       input,
		suggestions: suggestions,
		max:         30,
		styles:      fuzzy_search.NewStyles(),
		historyMode: historyFuzzy,
	}
	return fzf, fzf.editStatus
}
