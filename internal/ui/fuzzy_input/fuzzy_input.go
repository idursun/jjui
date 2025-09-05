package fuzzy_input

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/sahilm/fuzzy"
)

type suggestMode int

const (
	suggestOff suggestMode = iota
	suggestFuzzy
	suggestRegex
)

const ctrl_r = "ctrl+r"

type FuzzyInputModel struct {
	suggestions []string
	max         int
	styles      styles
	suggestMode suggestMode
	fuzzyView   *fuzzy_search.Model
}

type initMsg struct{}

func newCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func (fzf *FuzzyInputModel) Init() tea.Cmd {
	return newCmd(initMsg{})
}

func (fzf *FuzzyInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (fzf *FuzzyInputModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case ctrl_r == msg.String():
		switch fzf.suggestMode {
		case suggestOff:
			fzf.suggestMode = suggestFuzzy
			return nil
		case suggestFuzzy:
			fzf.suggestMode = suggestRegex
			return nil
		case suggestRegex:
			fzf.suggestMode = suggestOff
			//fzf.fuzzyView.MoveCursor(0)
			return nil
		}
		//case key.Matches(msg, fzf.input.KeyMap.AcceptSuggestion) && fzf.hasSuggestions():
		//fzf.input.SetValue(suggestion)
		//fzf.input.CursorEnd()
		return nil
		//case key.Matches(msg, km.Preview.ScrollUp, fzf.input.KeyMap.PrevSuggestion):
		//	fzf.moveCursor(1)
		//	return skipSearch
		//case key.Matches(msg, km.Preview.ScrollDown, fzf.input.KeyMap.NextSuggestion):
		//	fzf.moveCursor(-1)
		//return skipSearch
	case !fzf.suggestEnabled():
		return nil
	}
	return nil
}

func (fzf *FuzzyInputModel) suggestEnabled() bool {
	return fzf.suggestMode != suggestOff
}

func (fzf *FuzzyInputModel) hasSuggestions() bool {
	return fzf.suggestEnabled() && len(fzf.fuzzyView.Matches) > 0
}

func (fzf *FuzzyInputModel) search(input string) {
	input = strings.TrimSpace(input)
	fzf.fuzzyView.Cursor = 0
	fzf.fuzzyView.Matches = fuzzy.Matches{}
	if len(input) == 0 {
		return
	}
	if fzf.suggestMode == suggestFuzzy {
		fzf.fuzzyView.Search(input)
	} else if fzf.suggestMode == suggestRegex {
		fzf.fuzzyView.SearchRegex(input)
	}
}

func (fzf *FuzzyInputModel) View() string {
	matches := len(fzf.fuzzyView.Matches)
	if matches == 0 {
		return ""
	}
	view := fzf.fuzzyView.View()
	title := fmt.Sprintf(
		"  %s of %s elements in history ",
		strconv.Itoa(matches),
		strconv.Itoa(fzf.fuzzyView.Source.Len()),
	)
	title = fzf.styles.SelectedMatch.Render(title)
	return lipgloss.JoinVertical(0, title, view)
}

func (fzf *FuzzyInputModel) ShortHelp() []key.Binding {
	var shortHelp []key.Binding
	bind := func(keys string, help string) key.Binding {
		return key.NewBinding(key.WithKeys(keys), key.WithHelp(keys, help))
	}

	upDown := "ctrl+p/ctrl+n"

	moveOnHistory := bind(upDown, "move on history")
	moveOnSuggestions := bind(upDown, "move on suggest")

	switch fzf.suggestMode {
	case suggestOff:
		shortHelp = append(shortHelp, bind(ctrl_r, "suggest: off"), moveOnHistory)
	case suggestFuzzy:
		shortHelp = append(shortHelp, bind(ctrl_r, "suggest: fuzzy"), moveOnSuggestions)
	case suggestRegex:
		shortHelp = append(shortHelp, bind(ctrl_r, "suggest: regex"), moveOnSuggestions)
	}
	return shortHelp
}

func (fzf *FuzzyInputModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{fzf.ShortHelp()}
}

var _ fuzzy.Source = (*source)(nil)

type source struct {
	items []string
}

func (s source) String(i int) string {
	return s.items[i]
}

func (s source) Len() int {
	return len(s.items)
}

type styles struct {
	Dimmed        lipgloss.Style
	DimmedMatch   lipgloss.Style
	Selected      lipgloss.Style
	SelectedMatch lipgloss.Style
}
