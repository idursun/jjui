package fuzzy_search

import (
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
)

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
	if len(match.MatchedIndexes) == 0 {
		return lineStyle.Render(line)
	}

	var b strings.Builder
	last := 0
	for _, rng := range matchedRanges(match.MatchedIndexes) {
		start := min(rng[0], len(line))
		if start > last {
			b.WriteString(lineStyle.Render(line[last:start]))
		}

		_, size := utf8.DecodeRuneInString(line[start:])
		end := min(rng[1]+size, len(line))
		if end > start {
			b.WriteString(matchStyle.Render(line[start:end]))
		}
		last = end
	}
	if last < len(line) {
		b.WriteString(lineStyle.Render(line[last:]))
	}
	return b.String()
}

func matchedRanges(in []int) [][2]int {
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
