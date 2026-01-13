package fuzzy_search

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/rivo/uniseg"
	"github.com/sahilm/fuzzy"
)

type Styles struct {
	Dimmed        lipgloss.Style
	DimmedMatch   lipgloss.Style
	Selected      lipgloss.Style
	SelectedMatch lipgloss.Style
}

type Model interface {
	fuzzy.Source
	common.ImmediateModel
	Max() int
	Matches() fuzzy.Matches
	SelectedMatch() int
	Styles() Styles
}

type SearchMsg struct {
	Input   string
	Pressed tea.KeyMsg
}

func NewStyles() Styles {
	return Styles{
		Dimmed:        common.DefaultPalette.Get("status dimmed"),
		DimmedMatch:   common.DefaultPalette.Get("status shortcut"),
		Selected:      common.DefaultPalette.Get("selected"),
		SelectedMatch: common.DefaultPalette.Get("status title"),
	}
}

func Search(input string, key tea.KeyMsg) tea.Cmd {
	return func() tea.Msg {
		return SearchMsg{
			Input:   input,
			Pressed: key,
		}
	}
}

func SelectedMatch(model Model) string {
	idx := model.SelectedMatch()
	matches := model.Matches()
	n := len(matches)
	if idx < 0 || idx >= n {
		return ""
	}
	m := matches[idx]
	return fmt.Sprintf("'%s'", model.String(m.Index))
}

func View(fzf Model) string {
	shown := []string{}
	max := fzf.Max()
	styles := fzf.Styles()
	selected := fzf.SelectedMatch()
	for i, match := range fzf.Matches() {
		if i == max {
			break
		}
		sel := " "
		selStyle := styles.SelectedMatch
		lineStyle := styles.Dimmed
		matchStyle := styles.DimmedMatch

		entry := fzf.String(match.Index)
		if i == selected {
			sel = "◆"
			lineStyle = styles.Selected
			matchStyle = styles.SelectedMatch
		}

		entry = HighlightMatched(entry, match, lineStyle, matchStyle)
		shown = append(shown, selStyle.Render(sel)+" "+entry)
	}
	slices.Reverse(shown)
	entries := lipgloss.JoinVertical(0, shown...)
	return entries
}

type RefinedSource struct {
	Source  fuzzy.Source
	matches fuzzy.Matches
}

// each space on input creates a refined search: filtering on previous matches
func (fzf *RefinedSource) Search(input string, max int) fuzzy.Matches {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		fzf.matches = fuzzy.Matches{}
		flen := fzf.Source.Len()
		for i := range max {
			if i == flen {
				return fzf.matches
			}
			fzf.matches = append(fzf.matches, fuzzy.Match{
				Index: i,
				Str:   fzf.Source.String(i),
			})
		}
		return fzf.matches
	}
	for i, input := range strings.Fields(input) {
		if i == 0 {
			fzf.matches = fuzzy.FindFrom(input, fzf.Source)
		} else {
			matches := fuzzy.Matches{}
			for _, m := range fuzzy.FindFrom(input, fzf) {
				prev := fzf.matches[m.Index]
				matches = append(matches, fuzzy.Match{
					Str:            m.Str,
					MatchedIndexes: m.MatchedIndexes,
					Score:          m.Score,
					Index:          prev.Index,
				})
			}
			fzf.matches = matches
		}
	}
	return fzf.matches
}

func (fzf *RefinedSource) Len() int {
	return len(fzf.matches)
}

func (fzf *RefinedSource) String(i int) string {
	match := fzf.matches[i]
	return fzf.Source.String(match.Index)
}

// Adapted from gum/filter.go
func HighlightMatched(line string, match fuzzy.Match, lineStyle lipgloss.Style, matchStyle lipgloss.Style) string {
	var ranges []lipgloss.Range
	for _, rng := range matchedRanges(match.MatchedIndexes) {
		start, stop := bytePosToVisibleCharPos(match.Str, rng)
		ranges = append(ranges, lipgloss.NewRange(start, stop+1, matchStyle))
	}
	return lineStyle.Render(lipgloss.StyleRanges(line, ranges...))
}

// copied from gum/filter.go (MIT Licensed)
func matchedRanges(in []int) [][2]int {
	if len(in) == 0 {
		return [][2]int{}
	}
	current := [2]int{in[0], in[0]}
	if len(in) == 1 {
		return [][2]int{current}
	}
	var out [][2]int
	for i := 1; i < len(in); i++ {
		if in[i] == current[1]+1 {
			current[1] = in[i]
		} else {
			out = append(out, current)
			current = [2]int{in[i], in[i]}
		}
	}
	out = append(out, current)
	return out
}

// copied from gum/filter.go (MIT Licensed)
func bytePosToVisibleCharPos(str string, rng [2]int) (int, int) {
	bytePos, byteStart, byteStop := 0, rng[0], rng[1]
	pos, start, stop := 0, 0, 0
	gr := uniseg.NewGraphemes(str)
	for byteStart > bytePos {
		if !gr.Next() {
			break
		}
		bytePos += len(gr.Str())
		pos += max(1, gr.Width())
	}
	start = pos
	for byteStop > bytePos {
		if !gr.Next() {
			break
		}
		bytePos += len(gr.Str())
		pos += max(1, gr.Width())
	}
	stop = pos
	return start, stop
}
