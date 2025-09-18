package fuzzy_search

import (
	"regexp"
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

type Model struct {
	Source  fuzzy.Source
	Matches fuzzy.Matches
	max     int
	Cursor  int
	styles  Styles
}

func NewModel(source fuzzy.Source, max int) *Model {
	return &Model{
		Source: source,
		max:    max,
		Cursor: 0,
		styles: NewStyles(),
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	return m, nil
}

func (m *Model) View() string {
	var shown []string
	selected := m.Cursor
	for i, match := range m.Matches {
		if i == m.max {
			break
		}
		sel := " "
		selStyle := m.styles.SelectedMatch
		lineStyle := m.styles.Dimmed
		matchStyle := m.styles.DimmedMatch

		entry := m.Source.String(match.Index)
		if i == selected {
			sel = "◆"
			lineStyle = m.styles.Selected
			matchStyle = m.styles.SelectedMatch
		}

		entry = HighlightMatched(entry, match, lineStyle, matchStyle)
		shown = append(shown, selStyle.Render(sel)+" "+entry)
	}
	slices.Reverse(shown)
	entries := lipgloss.JoinVertical(0, shown...)
	return entries
}

func (m *Model) Search(input string) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		m.Matches = fuzzy.Matches{}
		n := m.Source.Len()
		for i := range m.max {
			if i == n {
				break
			}
			m.Matches = append(m.Matches, fuzzy.Match{
				Index: i,
				Str:   m.Source.String(i),
			})
		}
		return
	}

	m.Matches = fuzzy.FindFrom(input, m.Source)
}

func (m *Model) SearchRegex(input string) fuzzy.Matches {
	matches := fuzzy.Matches{}
	re, err := regexp.CompilePOSIX(input)
	if err != nil {
		return matches
	}
	for i := range m.Source.Len() {
		str := m.Source.String(i)
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

func (m *Model) SelectedMatch() string {
	idx := m.Cursor
	matches := m.Matches
	n := len(matches)
	if idx < 0 || idx >= n {
		return ""
	}
	match := matches[idx]
	return m.Source.String(match.Index)
}

func (m *Model) MoveCursor(delta int) {
	n := len(m.Matches)
	if n == 0 {
		m.Cursor = 0
		return
	}
	m.Cursor += delta
	if m.Cursor < 0 {
		m.Cursor = 0
	} else if m.Cursor >= n {
		m.Cursor = n - 1
	}
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

// HighlightMatched Adapted from gum/filter.go
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
