package fuzzy_input

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/sahilm/fuzzy"
)

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

func (fzf *model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case initMsg:
		fzf.search("")
	case fuzzy_search.SearchMsg:
		fzf.search(msg.Input)
	case intents.Intent:
		return fzf.handleIntent(msg)
	}
	return nil
}

func (fzf *model) handleIntent(intent intents.Intent) tea.Cmd {
	switch intent := intent.(type) {
	case intents.SuggestCycle:
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
	case intents.SuggestNavigate:
		fzf.moveCursor(intent.Delta)
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

func (fzf *model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	content := fzf.viewContent()
	if content == "" {
		return
	}
	_, h := lipgloss.Size(content)
	rect := layout.Rect(box.R.Min.X, box.R.Max.Y-h, box.R.Dx(), h)
	dl.AddDraw(rect, content, render.ZFuzzyInput)
}

func (fzf *model) viewContent() string {
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

func NewModel(input *textinput.Model, suggestions []string) fuzzy_search.Model {
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
	return fzf
}
