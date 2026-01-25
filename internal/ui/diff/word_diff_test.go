package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"hello world", []string{"hello", " ", "world"}},
		{"foo_bar", []string{"foo_bar"}},
		{"a.b", []string{"a", ".", "b"}},
		{"x + y", []string{"x", " ", "+", " ", "y"}},
		{"  spaces  ", []string{"  ", "spaces", "  "}},
		{"", nil},
		{"abc123", []string{"abc123"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := tokenize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComputeWordDiff_SimpleChange(t *testing.T) {
	oldLine := &DiffLine{Content: "hello world"}
	newLine := &DiffLine{Content: "hello there"}

	ComputeWordDiff(oldLine, newLine)

	// "hello " is common, "world" is removed, "there" is added
	assert.NotEmpty(t, oldLine.Segments)
	assert.NotEmpty(t, newLine.Segments)

	// Verify the changed part is highlighted
	hasHighlightedOld := false
	hasHighlightedNew := false
	for _, seg := range oldLine.Segments {
		if seg.Highlight && seg.Text == "world" {
			hasHighlightedOld = true
		}
	}
	for _, seg := range newLine.Segments {
		if seg.Highlight && seg.Text == "there" {
			hasHighlightedNew = true
		}
	}
	assert.True(t, hasHighlightedOld, "old line should highlight 'world'")
	assert.True(t, hasHighlightedNew, "new line should highlight 'there'")
}

func TestComputeWordDiff_NoChange(t *testing.T) {
	oldLine := &DiffLine{Content: "same content"}
	newLine := &DiffLine{Content: "same content"}

	ComputeWordDiff(oldLine, newLine)

	// All segments should be non-highlighted
	for _, seg := range oldLine.Segments {
		assert.False(t, seg.Highlight)
	}
	for _, seg := range newLine.Segments {
		assert.False(t, seg.Highlight)
	}
}

func TestComputeWordDiff_CompleteChange(t *testing.T) {
	oldLine := &DiffLine{Content: "foo"}
	newLine := &DiffLine{Content: "bar"}

	ComputeWordDiff(oldLine, newLine)

	// Everything should be highlighted
	for _, seg := range oldLine.Segments {
		if seg.Text != "" {
			assert.True(t, seg.Highlight)
		}
	}
	for _, seg := range newLine.Segments {
		if seg.Text != "" {
			assert.True(t, seg.Highlight)
		}
	}
}

func TestComputeWordDiff_NilLines(t *testing.T) {
	// Should not panic
	ComputeWordDiff(nil, nil)
	ComputeWordDiff(&DiffLine{Content: "test"}, nil)
	ComputeWordDiff(nil, &DiffLine{Content: "test"})
}

func TestComputeWordDiffForHunk_AdjacentLines(t *testing.T) {
	hunk := &Hunk{
		Lines: []DiffLine{
			{Type: LineContext, Content: "context"},
			{Type: LineRemoved, Content: "old value"},
			{Type: LineAdded, Content: "new value"},
			{Type: LineContext, Content: "more context"},
		},
	}

	ComputeWordDiffForHunk(hunk)

	// The removed and added lines should have segments
	assert.NotEmpty(t, hunk.Lines[1].Segments)
	assert.NotEmpty(t, hunk.Lines[2].Segments)
}

func TestComputeWordDiffForHunk_MultipleChanges(t *testing.T) {
	hunk := &Hunk{
		Lines: []DiffLine{
			{Type: LineRemoved, Content: "line 1"},
			{Type: LineRemoved, Content: "line 2"},
			{Type: LineAdded, Content: "line A"},
			{Type: LineAdded, Content: "line B"},
		},
	}

	ComputeWordDiffForHunk(hunk)

	// All lines should have segments (paired: 1-A, 2-B)
	for _, line := range hunk.Lines {
		assert.NotEmpty(t, line.Segments)
	}
}

func TestComputeWordDiffForHunk_UnpairedLines(t *testing.T) {
	hunk := &Hunk{
		Lines: []DiffLine{
			{Type: LineRemoved, Content: "only removed"},
		},
	}

	ComputeWordDiffForHunk(hunk)

	// Unpaired line should be fully highlighted
	assert.Len(t, hunk.Lines[0].Segments, 1)
	assert.True(t, hunk.Lines[0].Segments[0].Highlight)
}
