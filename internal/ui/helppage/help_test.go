package helppage

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/test"
)

func TestSearchEntriesIncludesModeGroupResults(t *testing.T) {
	model := &Model{
		styles: styles{
			title:    lipgloss.NewStyle(),
			dimmed:   lipgloss.NewStyle(),
			text:     lipgloss.NewStyle(),
			shortcut: lipgloss.NewStyle(),
			border:   lipgloss.NewStyle(),
		},
	}
	model.searchQuery = "mode"

	modeEntry := helpEntry{
		view:        "Mode Entry",
		search:      "mode entry",
		isModeEntry: true,
	}
	childEntry := helpEntry{
		view:   "Child Entry",
		search: "child entry",
	}

	lines := model.searchEntries(5, []helpEntry{modeEntry, childEntry})
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}
	if got := lines[0]; got != "Search: mode" {
		t.Fatalf("expected search header %q, got %q", "Search: mode", got)
	}
	if got := lines[2]; got != modeEntry.view {
		t.Fatalf("expected mode entry to be included, got %q", got)
	}
	if got := lines[3]; got != childEntry.view {
		t.Fatalf("expected child entry to be included, got %q", got)
	}
}

func TestSearchEntriesShowsNoMatchesMessage(t *testing.T) {
	model := &Model{
		styles: styles{
			title:  lipgloss.NewStyle(),
			dimmed: lipgloss.NewStyle(),
		},
	}
	model.searchQuery = "unknown"

	lines := model.searchEntries(4, []helpEntry{
		{view: "Mode Entry", search: "mode entry", isModeEntry: true},
	})
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}

	expected := "No matching help entries."
	if got := lines[2]; got != expected {
		t.Fatalf("expected %q message, got %q", expected, got)
	}
}

func TestUpdateSearchLifecycle(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))

	if model.searchActive {
		t.Fatalf("expected search to be inactive initially")
	}

	model.searchQuery = "existing"
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if cmd != nil {
		t.Fatalf("expected no command on search activation, got %T", cmd)
	}
	if !model.searchActive {
		t.Fatalf("expected search to become active after '/' key")
	}
	if model.searchQuery != "" {
		t.Fatalf("expected search query to reset, got %q", model.searchQuery)
	}

	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Fatalf("expected no command when typing in search, got %T", cmd)
	}
	if model.searchQuery != "a" {
		t.Fatalf("expected search query %q, got %q", "a", model.searchQuery)
	}

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if model.searchQuery != "ab" {
		t.Fatalf("expected search query %q, got %q", "ab", model.searchQuery)
	}

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if model.searchQuery != "a" {
		t.Fatalf("expected search query %q after backspace, got %q", "a", model.searchQuery)
	}

	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("expected no command on escape, got %T", cmd)
	}
	if model.searchActive {
		t.Fatalf("expected search to deactivate after escape")
	}
	if model.searchQuery != "" {
		t.Fatalf("expected search query to be cleared after escape, got %q", model.searchQuery)
	}
}
