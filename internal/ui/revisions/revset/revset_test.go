package revset

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestSignatureHelp(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ancestors", "ancestors("},
		{"ancestors(", "ancestors("},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			model := New("")
			model.Editing = true
			model.textInput.SetValue(test.input)
			m, _ := model.Update(tea.KeyLeft)
			assert.Contains(t, m.signatureHelp, test.expected)
		})
	}
}

func TestSuggestions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ancestors", "ancestors("},
		{"ancestors(visible_", "ancestors(visible_heads()"},
		{"author", "author("},
		{"author(m", "author(mine()"},
		{"author( m", "author( mine()"},
		{"present(@) | m", "present(@) | mine()"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			model := New("")
			model.Editing = true
			model.textInput.SetValue(test.input)
			m, _ := model.Update(tea.KeyLeft)
			suggestions := m.textInput.AvailableSuggestions()
			assert.Contains(t, suggestions, test.expected)
		})
	}
}
