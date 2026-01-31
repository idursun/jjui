package diff

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestNew_TrimsCarriageReturnsAndHandlesEmpty(t *testing.T) {
	model := New(nil, "", nil, "", "line1\r\nline2\r\n")
	assert.Equal(t, "line1\nline2", test.Stripped(test.RenderImmediate(model, 20, 5)))

	emptyModel := New(nil, "", nil, "", "")
	assert.Equal(t, "(empty)", test.Stripped(test.RenderImmediate(emptyModel, 10, 3)))
}

func TestScroll_AdjustsViewportOffset(t *testing.T) {
	content := "1\n2\n3\n4\n5\n"
	model := New(nil, "", nil, "", content)

	model.Scroll(2)
	assert.Equal(t, 2, model.startLine)

	model.Scroll(-1)
	assert.Equal(t, 1, model.startLine)
}

func TestUpdate_CancelReturnsClose(t *testing.T) {
	model := New(nil, "", nil, "", "content")
	model.keymap.Cancel = key.NewBinding(key.WithKeys("q"))

	var msgs []tea.Msg
	test.SimulateModel(model, test.Type("q"), func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})

	assert.Contains(t, msgs, common.CloseViewMsg{})
}

func TestSearch_FindsMatches(t *testing.T) {
	content := "hello world\nfoo bar\nhello again"
	model := New(nil, "", nil, "", content)

	// Perform search
	model.searchQuery = "hello"
	model.performSearch()

	assert.Len(t, model.searchMatches, 2)
	assert.Equal(t, 0, model.searchMatches[0].lineIdx)
	assert.Equal(t, 0, model.searchMatches[0].startCol)
	assert.Equal(t, 5, model.searchMatches[0].endCol)
	assert.Equal(t, 2, model.searchMatches[1].lineIdx)
	assert.Equal(t, 0, model.searchMatches[1].startCol)
}

func TestSearch_CaseInsensitive(t *testing.T) {
	content := "Hello World\nHELLO WORLD\nhello world"
	model := New(nil, "", nil, "", content)

	model.searchQuery = "hello"
	model.performSearch()

	assert.Len(t, model.searchMatches, 3)
}

func TestSearch_NavigateMatches(t *testing.T) {
	content := "match1\nmatch2\nmatch3"
	model := New(nil, "", nil, "", content)

	model.searchQuery = "match"
	model.performSearch()

	assert.Len(t, model.searchMatches, 3)
	assert.Equal(t, 0, model.currentMatch)

	model.jumpToNextMatch()
	assert.Equal(t, 1, model.currentMatch)

	model.jumpToNextMatch()
	assert.Equal(t, 2, model.currentMatch)

	model.jumpToPrevMatch()
	assert.Equal(t, 1, model.currentMatch)
}

func TestSearch_WrapAround(t *testing.T) {
	content := "match1\nmatch2\nmatch3"
	model := New(nil, "", nil, "", content)

	model.searchQuery = "match"
	model.performSearch()

	// Jump to last
	model.currentMatch = 2

	// Wrap forward
	model.jumpToNextMatch()
	assert.Equal(t, 0, model.currentMatch)

	// Wrap backward
	model.jumpToPrevMatch()
	assert.Equal(t, 2, model.currentMatch)
}

func TestSearch_ClearOnEscape(t *testing.T) {
	content := "hello world"
	model := New(nil, "", nil, "", content)
	model.keymap.Cancel = key.NewBinding(key.WithKeys("esc"))

	// Set up active search
	model.searchQuery = "hello"
	model.performSearch()
	assert.True(t, model.hasActiveSearch())

	// Clear with escape
	model.Update(tea.KeyMsg{Type: tea.KeyEsc})

	assert.False(t, model.hasActiveSearch())
	assert.Empty(t, model.searchMatches)
}

func TestSearch_MultipleMatchesOnSameLine(t *testing.T) {
	content := "hello hello hello"
	model := New(nil, "", nil, "", content)

	model.searchQuery = "hello"
	model.performSearch()

	assert.Len(t, model.searchMatches, 3)
	assert.Equal(t, 0, model.searchMatches[0].startCol)
	assert.Equal(t, 6, model.searchMatches[1].startCol)
	assert.Equal(t, 12, model.searchMatches[2].startCol)
}

func TestSearch_NoMatches(t *testing.T) {
	content := "hello world"
	model := New(nil, "", nil, "", content)

	model.searchQuery = "xyz"
	model.performSearch()

	assert.Empty(t, model.searchMatches)
}

func TestSearch_StartSearchMode(t *testing.T) {
	content := "hello world"
	model := New(nil, "", nil, "", content)
	model.keymap.DiffViewer.Search = key.NewBinding(key.WithKeys("/"))

	// Start search mode
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	assert.True(t, model.isSearching)
}

func TestSearch_NextMatchWithN(t *testing.T) {
	content := "match1\nmatch2\nmatch3"
	model := New(nil, "", nil, "", content)
	model.keymap.DiffViewer.NextHunk = key.NewBinding(key.WithKeys("n"))

	// Set up search
	model.searchQuery = "match"
	model.performSearch()
	assert.Equal(t, 0, model.currentMatch)

	// Press n to go to next match
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.Equal(t, 1, model.currentMatch)
}

func TestSearch_PrevMatchWithN(t *testing.T) {
	content := "match1\nmatch2\nmatch3"
	model := New(nil, "", nil, "", content)
	model.keymap.DiffViewer.PrevHunk = key.NewBinding(key.WithKeys("N"))

	// Set up search
	model.searchQuery = "match"
	model.performSearch()
	model.currentMatch = 2

	// Press N to go to previous match
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	assert.Equal(t, 1, model.currentMatch)
}
