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
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/sahilm/fuzzy"
)

const ctrl_r = "ctrl+r"

// var _ tea.Model = (*Model)(nil)
var _ fuzzy_search.FuzzySearchModel = (*Model)(nil)

type Model struct {
	suggestions []string
	input       textinput.Model
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

func (fzf *Model) Init() tea.Cmd {
	return tea.Batch(newCmd(initMsg{}), fzf.input.Focus())
}

func (fzf *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case actions.InvokeActionMsg:
		switch msg.Action.Id {
		case "fuzzy.up":
			fzf.moveCursor(1)
			return fzf, nil
		case "fuzzy.down":
			fzf.moveCursor(-1)
			return fzf, nil
		case "fuzzy.complete":
			suggestion := fuzzy_search.SelectedMatch(fzf)
			fzf.input.SetValue(suggestion)
			fzf.input.CursorEnd()
			return fzf, nil
		case "fuzzy.cycle_suggest_mode":
			fzf.CycleSuggestMode()
			return fzf, nil
		}
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

func (fzf *Model) CycleSuggestMode() {
	switch fzf.suggestMode {
	case config.SuggestModeOff:
		fzf.suggestMode = config.SuggestModeFuzzy
	case config.SuggestModeFuzzy:
		fzf.suggestMode = config.SuggestModeRegex
	case config.SuggestModeRegex:
		fzf.suggestMode = config.SuggestModeOff
		fzf.cursor = 0
		fzf.matches = nil
	}
}

func (fzf *Model) SetSuggestions(suggestions []string) {
	fzf.suggestions = suggestions
	fzf.input.SetSuggestions(suggestions)
}

func (fzf *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	//km := config.Current.GetKeyMap()
	//switch {
	//case key.Matches(msg, fzf.input.KeyMap.AcceptSuggestion) && fzf.hasSuggestions():
	//	suggestion := fuzzy_search.SelectedMatch(fzf)
	//	fzf.input.SetValue(suggestion)
	//	fzf.input.CursorEnd()
	//	return nil
	//}
	var cmd tea.Cmd
	fzf.input, cmd = fzf.input.Update(msg)
	fzf.search(fzf.input.Value())
	return cmd
}

func (fzf *Model) suggestEnabled() bool { return fzf.suggestMode != config.SuggestModeOff }

func (fzf *Model) hasSuggestions() bool {
	return fzf.suggestEnabled() && len(fzf.matches) > 0
}

func (fzf *Model) moveCursor(inc int) {
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

func (fzf *Model) Styles() fuzzy_search.Styles {
	return fzf.styles
}

func (fzf *Model) Max() int {
	return fzf.max
}

func (fzf *Model) Matches() fuzzy.Matches {
	return fzf.matches
}

func (fzf *Model) SelectedMatch() int {
	return fzf.cursor
}

func (fzf *Model) Len() int {
	return len(fzf.suggestions)
}

func (fzf *Model) String(i int) string {
	if len(fzf.suggestions) == 0 {
		return ""
	}
	return fzf.suggestions[i]
}

func (fzf *Model) search(input string) {
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

func (fzf *Model) searchRegex(input string) fuzzy.Matches {
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
		var indexes []int
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

func (fzf *Model) CompletionView() string {
	matches := len(fzf.matches)
	if matches == 0 {
		return ""
	}
	title := fmt.Sprintf(
		"  %s of %s elements in history ",
		strconv.Itoa(matches),
		strconv.Itoa(fzf.Len()),
	)
	title = fzf.styles.SelectedMatch.Render(title)
	view := fuzzy_search.View(fzf)
	return lipgloss.JoinVertical(0, title, view)
}

func (fzf *Model) View() string {
	inputView := fzf.input.View()
	return inputView
}

func (fzf *Model) ShortHelp() []key.Binding {
	var shortHelp []key.Binding
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

func (fzf *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{fzf.ShortHelp()}
}

func (fzf *Model) Focus() tea.Cmd {
	return fzf.input.Focus()
}

func (fzf *Model) Value() string {
	return fzf.input.Value()
}

func (fzf *Model) editStatus() (help.KeyMap, string) {
	return fzf, ""
}

func NewModel(input textinput.Model, suggestions []string) *Model {
	input.ShowSuggestions = false
	input.SetSuggestions([]string{})

	suggestMode, err := config.GetSuggestExecMode(config.Current)
	if err != nil {
		log.Fatal(err)
	}

	fzf := &Model{
		input:       input,
		suggestions: suggestions,
		max:         30,
		styles:      fuzzy_search.NewStyles(),
		suggestMode: suggestMode,
	}
	return fzf
}
