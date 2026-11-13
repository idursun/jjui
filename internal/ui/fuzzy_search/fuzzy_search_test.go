package fuzzy_search

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
	"github.com/stretchr/testify/assert"
)

type simpleSource []string

func (s simpleSource) Len() int {
	return len(s)
}

func (s simpleSource) String(i int) string {
	return s[i]
}

func TestRefinedSourceSearch_EmptyInputReturnsPrefix(t *testing.T) {
	src := &RefinedSource{Source: simpleSource{"one", "two", "three"}}

	matches := src.Search("", 2)

	if assert.Len(t, matches, 2) {
		assert.Equal(t, "one", matches[0].Str)
		assert.Equal(t, 0, matches[0].Index)
		assert.Equal(t, "two", matches[1].Str)
		assert.Equal(t, 1, matches[1].Index)
	}
}

func TestRefinedSourceSearch_RefinesAcrossTokens(t *testing.T) {
	src := &RefinedSource{Source: simpleSource{"abc", "bcd", "cde"}}

	matches := src.Search("ab cd", 5)

	assert.Empty(t, matches, "second token should filter previous matches, leaving none")
}

func TestHighlightMatched_AppliesStylesWithoutChangingWidth(t *testing.T) {
	line := "hello"
	match := fuzzy.Match{Str: line, MatchedIndexes: []int{1, 2}}

	rendered := HighlightMatched(line, match, lipgloss.NewStyle(), lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true))

	assert.Equal(t, lipgloss.Width(line), lipgloss.Width(rendered))
}
