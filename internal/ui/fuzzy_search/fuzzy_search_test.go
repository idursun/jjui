package fuzzy_search

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
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

type stubModel struct {
	matches  fuzzy.Matches
	strings  []string
	selected int
}

func (m stubModel) Max() int                                        { return len(m.matches) }
func (m stubModel) Matches() fuzzy.Matches                          { return m.matches }
func (m stubModel) SelectedMatch() int                              { return m.selected }
func (m stubModel) Styles() Styles                                  { return NewStyles() }
func (m stubModel) Init() tea.Cmd                                   { return nil }
func (m stubModel) Update(msg tea.Msg) tea.Cmd                      { return nil }
func (m stubModel) ViewRect(_ *render.DisplayContext, _ layout.Box) {}
func (m stubModel) Len() int                                        { return len(m.strings) }
func (m stubModel) String(i int) string {
	return m.strings[i]
}

func TestSelectedMatch_ReturnsQuotedMatch(t *testing.T) {
	model := stubModel{
		strings:  []string{"zero", "one"},
		matches:  fuzzy.Matches{{Index: 1, Str: "one"}},
		selected: 0,
	}

	assert.Equal(t, "'one'", SelectedMatch(model))
}

func TestSelectedMatch_OutOfRange(t *testing.T) {
	model := stubModel{
		strings:  []string{"zero"},
		matches:  fuzzy.Matches{},
		selected: 0,
	}

	assert.Equal(t, "", SelectedMatch(model))
}

func TestHighlightMatched_AppliesStylesWithoutChangingWidth(t *testing.T) {
	line := "hello"
	match := fuzzy.Match{Str: line, MatchedIndexes: []int{1, 2}}

	rendered := HighlightMatched(line, match, lipgloss.NewStyle(), lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true))

	assert.Equal(t, lipgloss.Width(line), lipgloss.Width(rendered))
}
