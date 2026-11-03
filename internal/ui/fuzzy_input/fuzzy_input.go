package fuzzy_input

import (
	"log"
	"regexp"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
	"github.com/sahilm/fuzzy"
)

type ItemClickedMsg struct {
	Index int
}

type itemScrollMsg struct {
	Delta      int
	Horizontal bool
}

func newItemScrollMsg(msg tea.MouseMsg) tea.Msg {
	delta, horizontal := render.WheelDelta(msg)
	return itemScrollMsg{Delta: delta, Horizontal: horizontal}
}

type Model struct {
	suggestions  []string
	input        *textinput.Model
	cursor       int
	max          int
	matches      fuzzy.Matches
	suggestMode  config.SuggestMode
	listRenderer *render.ListRenderer
	// set when the cursor moves, cleared after a frame so wheel scrolling sticks
	ensureCursorVisible bool
}

func (fzf *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case intents.Intent:
		fzf.handleIntent(msg)
	case ItemClickedMsg:
		fzf.cursor = msg.Index
		fzf.ensureCursorVisible = true
	case itemScrollMsg:
		if !msg.Horizontal {
			fzf.ensureCursorVisible = false
			fzf.listRenderer.StartLine = max(0, fzf.listRenderer.StartLine+msg.Delta)
		}
	}
	return nil
}

func (fzf *Model) handleIntent(intent intents.Intent) {
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
}

func (fzf *Model) suggestEnabled() bool { return fzf.suggestMode != config.SuggestModeOff }

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
	fzf.ensureCursorVisible = true
	if !fzf.suggestEnabled() {
		// update input.
		fzf.input.SetValue(fzf.String(n))
		fzf.input.CursorEnd()
	}
}

// Selected returns the highlighted suggestion, or "" when no suggestion is shown.
func (fzf *Model) Selected() string {
	if fzf.cursor < 0 || fzf.cursor >= min(len(fzf.matches), fzf.max) {
		return ""
	}
	return fzf.String(fzf.matches[fzf.cursor].Index)
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

func (fzf *Model) Search(input string) {
	input = strings.TrimSpace(input)
	fzf.cursor = 0
	fzf.matches = fuzzy.Matches{}
	fzf.listRenderer.StartLine = 0
	fzf.ensureCursorVisible = true
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

// ViewRect draws the matches bottom-up at the bottom of box: the best match
// on the last row and the title above the list.
func (fzf *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	n := min(len(fzf.matches), fzf.max)
	listHeight := min(n, box.R.Dy()-1)
	if listHeight <= 0 || box.R.Dx() <= 0 {
		return
	}

	background := theme.DefaultPalette.Get("status", "", "text", false)
	titleStyle := theme.DefaultPalette.Get("status", "", "title", false)
	dimmedStyle := theme.DefaultPalette.Get("status", "", "dimmed", false)
	dimmedMatchStyle := theme.DefaultPalette.Get("status", "", "shortcut", false)
	selectedStyle := theme.DefaultPalette.GetBlended("", "", "", true)

	height := listHeight + 1
	rect := layout.Rect(box.R.Min.X, box.R.Max.Y-height, box.R.Dx(), height)
	dl.AddFill(rect, ' ', background, render.ZFuzzyOverlay-1)

	titleBox, listBox := layout.NewBox(rect).CutTop(1)
	title := "  " + strconv.Itoa(len(fzf.matches)) + " of " + strconv.Itoa(fzf.Len()) + " elements in history "
	dl.Text(titleBox.R.Min.X, titleBox.R.Min.Y, render.ZFuzzyOverlay).Styled(title, titleStyle).Done()

	// the list renders top-down, so row i shows match n-1-i
	fzf.listRenderer.Render(
		dl,
		listBox,
		n,
		n-1-fzf.cursor,
		fzf.ensureCursorVisible,
		func(int) int { return 1 },
		func(dl *render.DisplayContext, row int, r layout.Rectangle) {
			index := n - 1 - row
			if index < 0 || index >= n {
				return
			}
			match := fzf.matches[index]
			marker, lineStyle, matchStyle := " ", dimmedStyle, dimmedMatchStyle
			if index == fzf.cursor {
				marker, lineStyle, matchStyle = "◆", selectedStyle, titleStyle
			}
			entry := fuzzy_search.HighlightMatched(fzf.String(match.Index), match, lineStyle, matchStyle)
			dl.AddDraw(r, titleStyle.Render(marker)+" "+entry, render.ZFuzzyOverlay)
		},
		func(row int, _ tea.Mouse) tea.Msg { return ItemClickedMsg{Index: n - 1 - row} },
	)
	fzf.listRenderer.RegisterScroll(dl, listBox)
	fzf.ensureCursorVisible = false
}

func New(input *textinput.Model, suggestions []string) *Model {
	input.ShowSuggestions = false
	input.SetSuggestions([]string{})

	suggestMode, err := config.GetSuggestExecMode(config.Current)
	if err != nil {
		log.Fatal(err)
	}

	fzf := &Model{
		input:        input,
		suggestions:  suggestions,
		max:          30,
		suggestMode:  suggestMode,
		listRenderer: render.NewListRenderer(newItemScrollMsg),
	}
	fzf.listRenderer.Z = render.ZFuzzyOverlay
	return fzf
}
