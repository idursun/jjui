package fuzzy_input

import (
	"fmt"
	"log"
	"regexp"
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

const ctrl_r = "ctrl+r"

type model struct {
	suggestions []string
	input       *textinput.Model
	cursor      int
	max         int
	matches     fuzzy.Matches
	styles      fuzzy_search.Styles
	suggestMode config.SuggestMode
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
		switch fzf.suggestMode {
		case config.SuggestModeOff:
			fzf.suggestMode = config.SuggestModeFuzzy
			return nil
		case config.SuggestModeFuzzy:
			fzf.suggestMode = config.SuggestModeRegex
			return nil
		case config.SuggestModeRegex:
			fzf.suggestMode = config.SuggestModeOff
			fzf.cursor = 0
			fzf.matches = nil
			return skipSearch
		}
	case key.Matches(msg, fzf.input.KeyMap.AcceptSuggestion) && fzf.hasSuggestions():
		suggestion := fuzzy_search.SelectedMatch(fzf)
		fzf.input.SetValue(suggestion)
		fzf.input.CursorEnd()
		return skipSearch
	case key.Matches(msg, km.Preview.ScrollUp, fzf.input.KeyMap.PrevSuggestion):
		fzf.moveCursor(1)
		return skipSearch
	case key.Matches(msg, km.Preview.ScrollDown, fzf.input.KeyMap.NextSuggestion):
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
	case !fzf.suggestEnabled():
		return skipSearch
	}
	return nil
}

func (fzf *model) suggestEnabled() bool { return fzf.suggestMode != config.SuggestModeOff }

func (fzf *model) hasSuggestions() bool {
	return fzf.suggestEnabled() && len(fzf.matches) > 0
}

func (fzf *model) moveCursor(inc int) {
	l := min(len(fzf.matches), fzf.max)
	if !fzf.suggestEnabled() {
		// move on complete history
		l = min(fzf.Len(), fzf.max)
	}
	n := fzf.cursor + inc
	if n < 0 {
		n = l - 1
	}
	if n >= l {
		n = 0
	}
	fzf.cursor = n
	if !fzf.suggestEnabled() {
		// update input.
		fzf.input.SetValue(fzf.String(n))
		fzf.input.CursorEnd()
	}
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
	if fzf.suggestMode == config.SuggestModeFuzzy {
		fzf.matches = fuzzy.FindFrom(input, fzf)
	} else if fzf.suggestMode == config.SuggestModeRegex {
		fzf.matches = fzf.searchRegex(input)
	}
}

func (fzf *model) searchRegex(input string) fuzzy.Matches {
	matches := fuzzy.Matches{}
	re, err := regexp.CompilePOSIX(input)
	if err != nil {
		return matches
	}
	for i := range fzf.Len() {
		str := fzf.String(i)
		loc := re.FindStringIndex(str)
		if loc == nil {
			continue
		}
		indexes := []int{}
		for i := range loc[1] - loc[0] {
			indexes = append(indexes, i+loc[0])
		}
		matches = append(matches, fuzzy.Match{
			Index:          i,
			Str:            str,
			MatchedIndexes: indexes,
		})
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
	shortHelp := []key.Binding{}
	bind := func(keys string, help string) key.Binding {
		return key.NewBinding(key.WithKeys(keys), key.WithHelp(keys, help))
	}

	upDown := "ctrl+p/ctrl+n"

	moveOnHistory := bind(upDown, "move on history")
	moveOnSuggestions := bind(upDown, "move on suggest")

	switch fzf.suggestMode {
	case config.SuggestModeOff:
		shortHelp = append(shortHelp, bind(ctrl_r, "suggest: off"), moveOnHistory)
	case config.SuggestModeFuzzy:
		shortHelp = append(shortHelp, bind(ctrl_r, "suggest: fuzzy"), moveOnSuggestions)
	case config.SuggestModeRegex:
		shortHelp = append(shortHelp, bind(ctrl_r, "suggest: regex"), moveOnSuggestions)
	}
	return shortHelp
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

	suggestMode, err := config.GetSuggestExecMode(config.Current)
	if err != nil {
		log.Fatal(err)
	}

	fzf := &model{
		input:       input,
		suggestions: suggestions,
		max:         30,
		styles:      fuzzy_search.NewStyles(),
		suggestMode: suggestMode,
	}
	return fzf, fzf.editStatus
}
