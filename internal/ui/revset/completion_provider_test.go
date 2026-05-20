package revset

import (
	"slices"
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
		{"ancestors", "ancestors("},
		{"ancestors(visible_", "visible_heads()"},
		{"author", "author("},
		{"ancestors(m", "mine()"},
		{"ancestors( m", "mine()"},
		{"present(@) | m", "mine()"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			provider := NewCompletionProvider(nil)
			suggestions := provider.GetCompletions(test.input)
			found := slices.Contains(suggestions, test.expected)
			assert.True(t, found, "Expected suggestion '%s' not found for input: '%s'", test.expected, test.input)
		})
	}
}

func TestGetCompletionItems_FunctionArgumentCompletions(t *testing.T) {
	provider := NewCompletionProvider(nil)
	remoteListCalls := 0
	provider.Load(func(args []string) ([]byte, error) {
		switch args[0] {
		case "bookmark", "tag":
			return nil, nil
		case "git":
			remoteListCalls++
			return []byte("origin\nupstream\n"), nil
		default:
			return nil, nil
		}
	})

	tests := []struct {
		name     string
		input    string
		expected string
		kind     CompletionKind
	}{
		{
			name:     "freeform pattern suggests matching named argument",
			input:    "untracked_remote_bookmarks(re",
			expected: "remote=",
			kind:     KindArgument,
		},
		{
			name:     "second positional slot suggests matching named argument",
			input:    "untracked_remote_bookmarks(foo, re",
			expected: "remote=",
			kind:     KindArgument,
		},
		{
			name:     "second positional slot suggests remote names",
			input:    "untracked_remote_bookmarks(foo, o",
			expected: "origin",
			kind:     KindRemote,
		},
		{
			name:     "named remote value suggests remote names",
			input:    "untracked_remote_bookmarks(foo, remote=o",
			expected: "origin",
			kind:     KindRemote,
		},
		{
			name:     "empty named remote value suggests all remote names",
			input:    "untracked_remote_bookmarks(foo, remote=",
			expected: "origin",
			kind:     KindRemote,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := provider.GetCompletionItems(test.input, nil)
			index := slices.IndexFunc(items, func(item CompletionItem) bool {
				return item.InsertText == test.expected && item.Kind == test.kind
			})
			assert.NotEqual(t, -1, index, "expected completion %q in %+v", test.expected, items)
		})
	}
	provider.GetCompletionItems("untracked_remote_bookmarks(foo, remote=u", nil)
	assert.Equal(t, 1, remoteListCalls, "remote completions should be cached after first load")
}

func TestGetCompletionItems_LoadRefreshesRemoteCompletions(t *testing.T) {
	provider := NewCompletionProvider(nil)

	provider.Load(func(args []string) ([]byte, error) {
		switch args[0] {
		case "bookmark", "tag":
			return nil, nil
		case "git":
			return []byte("origin\n"), nil
		default:
			return nil, nil
		}
	})
	assert.Contains(t, completionInsertTexts(provider.GetCompletionItems("remote_bookmarks(remote=o", nil)), "origin")

	provider.Load(func(args []string) ([]byte, error) {
		switch args[0] {
		case "bookmark", "tag":
			return nil, nil
		case "git":
			return []byte("upstream\n"), nil
		default:
			return nil, nil
		}
	})
	items := provider.GetCompletionItems("remote_bookmarks(remote=u", nil)
	assert.Contains(t, completionInsertTexts(items), "upstream")
	assert.NotContains(t, completionInsertTexts(items), "origin")
}

func TestGetCompletionItems_FreeformArgumentDoesNotSuggestRevsets(t *testing.T) {
	provider := NewCompletionProvider(nil)
	items := provider.GetCompletionItems("author(m", nil)
	assert.Empty(t, items)
}

func TestGetCompletionItems_DoesNotSuggestAlreadyUsedNamedArgument(t *testing.T) {
	provider := NewCompletionProvider(nil)
	items := provider.GetCompletionItems("remote_bookmarks(remote=origin, ", nil)
	assert.NotContains(t, completionInsertTexts(items), "remote=")
}

func completionInsertTexts(items []CompletionItem) []string {
	texts := make([]string, len(items))
	for i, item := range items {
		texts[i] = item.InsertText
	}
	return texts
}
