package source

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFunctionSource(t *testing.T) {
	s := FunctionSource{}
	items, err := s.Fetch(nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, items)

	for _, item := range items {
		assert.Equal(t, KindFunction, item.Kind)
		assert.NotEmpty(t, item.Name)
		assert.NotEmpty(t, item.SignatureHelp)
	}

	// Check a known function exists
	found := false
	for _, item := range items {
		if item.Name == "ancestors" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected 'ancestors' function")
}

func TestAliasSource(t *testing.T) {
	aliases := map[string]string{
		"my_alias()":  "mine()",
		"param(x, y)": "ancestors(x) & descendants(y)",
		"simple":      "trunk()",
	}
	s := AliasSource{Aliases: aliases}
	items, err := s.Fetch(nil)
	assert.NoError(t, err)
	assert.Len(t, items, 3)

	for _, item := range items {
		assert.Equal(t, KindAlias, item.Kind)
		assert.NotEmpty(t, item.SignatureHelp)
	}

	names := map[string]bool{}
	for _, item := range items {
		names[item.Name] = true
	}
	assert.True(t, names["my_alias"])
	assert.True(t, names["param"])
	assert.True(t, names["simple"])
}

func TestHistorySource(t *testing.T) {
	entries := []string{"trunk()", "mine()", "ancestors(@)"}
	s := HistorySource{Entries: entries}
	items, err := s.Fetch(nil)
	assert.NoError(t, err)
	assert.Len(t, items, 3)

	for i, item := range items {
		assert.Equal(t, KindHistory, item.Kind)
		assert.Equal(t, entries[i], item.Name)
	}
}

func TestHistorySourceEmpty(t *testing.T) {
	s := HistorySource{}
	items, err := s.Fetch(nil)
	assert.NoError(t, err)
	assert.Empty(t, items)
}

func TestBookmarkSource(t *testing.T) {
	mockRunner := func(args []string) ([]byte, error) {
		return []byte(`main;.;false;false;false;abc123
main;origin;true;false;false;abc123
feature;.;false;false;false;def456
`), nil
	}

	s := BookmarkSource{}
	items, err := s.Fetch(mockRunner)
	assert.NoError(t, err)

	names := make([]string, len(items))
	for i, item := range items {
		assert.Equal(t, KindBookmark, item.Kind)
		names[i] = item.Name
	}
	assert.Contains(t, names, "main")
	assert.Contains(t, names, "main@origin")
	assert.Contains(t, names, "feature")
}

func TestBookmarkSourceError(t *testing.T) {
	mockRunner := func(args []string) ([]byte, error) {
		return nil, fmt.Errorf("command failed")
	}

	s := BookmarkSource{}
	items, err := s.Fetch(mockRunner)
	assert.Error(t, err)
	assert.Nil(t, items)
}

func TestTagSource(t *testing.T) {
	mockRunner := func(args []string) ([]byte, error) {
		return []byte("v1.0.0\nv1.1.0\nv2.0.0\n"), nil
	}

	s := TagSource{}
	items, err := s.Fetch(mockRunner)
	assert.NoError(t, err)
	assert.Len(t, items, 3)

	for _, item := range items {
		assert.Equal(t, KindTag, item.Kind)
	}
	assert.Equal(t, "v1.0.0", items[0].Name)
	assert.Equal(t, "v1.1.0", items[1].Name)
	assert.Equal(t, "v2.0.0", items[2].Name)
}

func TestTagSourceError(t *testing.T) {
	mockRunner := func(args []string) ([]byte, error) {
		return nil, fmt.Errorf("command failed")
	}

	s := TagSource{}
	items, err := s.Fetch(mockRunner)
	assert.Error(t, err)
	assert.Nil(t, items)
}

func TestParseTagListOutput(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"v1.0.0\nv1.1.0\n", []string{"v1.0.0", "v1.1.0"}},
		{"  v1.0.0  \n\n  v2.0.0  \n", []string{"v1.0.0", "v2.0.0"}},
		{"", nil},
		{"\n\n\n", nil},
		{"single-tag", []string{"single-tag"}},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := ParseTagListOutput(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestFetchAll(t *testing.T) {
	items := FetchAll(nil, FunctionSource{}, HistorySource{Entries: []string{"test"}})
	assert.NotEmpty(t, items)

	// Last item should be the history entry
	last := items[len(items)-1]
	assert.Equal(t, "test", last.Name)
	assert.Equal(t, KindHistory, last.Kind)
}

func TestFetchAllSkipsFailures(t *testing.T) {
	failRunner := func(args []string) ([]byte, error) {
		return nil, fmt.Errorf("fail")
	}

	// BookmarkSource and TagSource will fail, but FunctionSource and HistorySource should succeed
	items := FetchAll(failRunner,
		FunctionSource{},
		BookmarkSource{},
		TagSource{},
		HistorySource{Entries: []string{"test"}},
	)

	// Should have functions + history, but no bookmarks or tags
	hasFunction := false
	hasHistory := false
	hasBookmark := false
	hasTag := false
	for _, item := range items {
		switch item.Kind {
		case KindFunction:
			hasFunction = true
		case KindHistory:
			hasHistory = true
		case KindBookmark:
			hasBookmark = true
		case KindTag:
			hasTag = true
		}
	}
	assert.True(t, hasFunction)
	assert.True(t, hasHistory)
	assert.False(t, hasBookmark)
	assert.False(t, hasTag)
}

func TestBaseFunctions(t *testing.T) {
	fns := BaseFunctions()
	assert.NotEmpty(t, fns)
	// Verify it's a copy
	fns[0].Name = "modified"
	original := BaseFunctions()
	assert.NotEqual(t, "modified", original[0].Name)
}
