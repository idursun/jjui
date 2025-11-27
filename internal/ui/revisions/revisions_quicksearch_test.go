package revisions

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/stretchr/testify/assert"
)

// mockNonFocusableOperation is a mock operation that is never focused, editing, or overlay
type mockNonFocusableOperation struct{}

func (m *mockNonFocusableOperation) Render(commit *jj.Commit, renderPosition int) string {
	return ""
}

func (m *mockNonFocusableOperation) Name() string {
	return "mock"
}

func (m *mockNonFocusableOperation) Init() tea.Cmd {
	return nil
}

func (m *mockNonFocusableOperation) Update(msg tea.Msg) tea.Cmd {
	return nil
}

func (m *mockNonFocusableOperation) View() string {
	return ""
}

func (m *mockNonFocusableOperation) IsFocused() bool {
	return false
}

func (m *mockNonFocusableOperation) IsEditing() bool {
	return false
}

func (m *mockNonFocusableOperation) IsOverlay() bool {
	return false
}

// TestQuickSearch_EnterKeyClearsSearch tests that Enter clears the search
func TestQuickSearch_EnterKeyClearsSearch(t *testing.T) {
	model := &Model{
		quickSearch: "test",
		renderer:    newRevisionListRenderer(nil, nil),
		op:          &mockNonFocusableOperation{},
		rows:        []parser.Row{{Commit: &jj.Commit{ChangeId: "test123"}}},
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	cmd := model.internalUpdate(msg)

	assert.Equal(t, "", model.quickSearch, "KeyEnter should clear quicksearch")
	assert.Nil(t, cmd)
}

// TestQuickSearch_EscapeKeyClearsSearch tests that Escape clears the search
func TestQuickSearch_EscapeKeyClearsSearch(t *testing.T) {
	model := &Model{
		quickSearch: "test",
		renderer:    newRevisionListRenderer(nil, nil),
		op:          &mockNonFocusableOperation{},
		rows:        []parser.Row{{Commit: &jj.Commit{ChangeId: "test123"}}},
	}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := model.internalUpdate(msg)

	assert.Equal(t, "", model.quickSearch, "KeyEsc should clear quicksearch")
	assert.Nil(t, cmd)
}

// TestQuickSearch_EnterWithEmptySearch tests that Enter with empty search does nothing special
func TestQuickSearch_EnterWithEmptySearch(t *testing.T) {
	model := &Model{
		quickSearch: "",
		renderer:    newRevisionListRenderer(nil, nil),
		rows:        []parser.Row{{Commit: &jj.Commit{ChangeId: "test123"}}},
		op:          &mockNonFocusableOperation{},
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_ = model.internalUpdate(msg)

	// When quickSearch is empty, Enter should not be handled by the quicksearch logic
	// The cmd might be nil or something else depending on other handlers
	assert.Equal(t, "", model.quickSearch)
}
