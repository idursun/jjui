package revset

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLastToken(t *testing.T) {
	provider := NewCompletionProvider(nil)
	tests := []struct {
		input         string
		expectedIndex int
		expectedToken string
	}{
		{"ancestors", 0, "ancestors"},
		{"ancestors(", 10, ""},
		{"author(m", 7, "m"},
		{"present(@) | m", 13, "m"},
		{"author( mine", 8, "mine"},
		{"", 0, ""},
		{"author_date(123) & ", 19, ""},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			index, token := provider.GetLastToken(test.input)
			assert.Equal(t, test.expectedIndex, index, "Index mismatch for input: %s", test.input)
			assert.Equal(t, test.expectedToken, token, "Token mismatch for input: %s", test.input)
		})
	}
}

func TestSignatureHelp(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"ancestors(", true},
		{"mine(", true},
		{"madeupfunction(", false},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			provider := NewCompletionProvider(nil)
			help := provider.GetSignatureHelp(test.input)
			assert.Equal(t, test.expected, help != "")
			if test.expected {
				assert.Contains(t, help, test.input)
			}
		})
	}
}

func TestGetCompletions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ancestors", "ancestors"},
		{"ancestors(visible_", "visible_heads()"},
		{"author", "author"},
		{"author(m", "mine()"},
		{"author( m", "mine()"},
		{"present(@) | m", "mine()"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			provider := NewCompletionProvider(nil)
			suggestions := provider.GetCompletions(test.input)
			found := false
			for _, suggestion := range suggestions {
				if suggestion == test.expected {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected suggestion '%s' not found for input: '%s'", test.expected, test.input)
		})
	}
}
