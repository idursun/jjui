package fuzzy_files

import (
	"testing"

	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/fuzzy_search"
	"github.com/sahilm/fuzzy"
	"github.com/stretchr/testify/assert"
)

func TestUpdateRevSet_WithPath(t *testing.T) {
	model := &fuzzyFiles{
		revset: "all()",
		files:  []string{"file1.txt", "path/to/file2.go", "special file.txt"},
		styles: fuzzy_search.NewStyles(),
	}

	// a match being selected
	model.matches = fuzzy.Matches{
		{Index: 1, Str: "path/to/file2.go"},
	}
	model.cursor = 0

	cmd := model.updateRevSet()

	// get the UpdateRevSet message
	msg := cmd()
	updateMsg, ok := msg.(common.UpdateRevSetMsg)
	assert.True(t, ok)

	// wrapped in single quotes by SelectedMatch in fuzzy_search
	assert.Equal(t, "files('path/to/file2.go')", string(updateMsg))
}

func TestUpdateRevSet_WithPathContainingSpaces(t *testing.T) {
	model := &fuzzyFiles{
		revset: "all()",
		files:  []string{"file with spaces.txt"},
		styles: fuzzy_search.NewStyles(),
	}

	model.matches = fuzzy.Matches{
		{Index: 0, Str: "file with spaces.txt"},
	}
	model.cursor = 0

	cmd := model.updateRevSet()
	msg := cmd()
	updateMsg, ok := msg.(common.UpdateRevSetMsg)
	assert.True(t, ok)

	// the whole path wrapped in single quotes
	assert.Equal(t, "files('file with spaces.txt')", string(updateMsg))
}

func TestUpdateRevSet_WithPathContainingBraces(t *testing.T) {
	model := &fuzzyFiles{
		revset: "all()",
		files:  []string{"file{with}braces.txt"},
		styles: fuzzy_search.NewStyles(),
	}

	model.matches = fuzzy.Matches{
		{Index: 0, Str: "file{with}braces.txt"},
	}
	model.cursor = 0

	cmd := model.updateRevSet()
	msg := cmd()
	updateMsg, ok := msg.(common.UpdateRevSetMsg)
	assert.True(t, ok)

	// braces should be preserved and the whole path wrapped in single quotes
	assert.Equal(t, "files('file{with}braces.txt')", string(updateMsg))
}

func TestUpdateRevSet_NoPath(t *testing.T) {
	model := &fuzzyFiles{
		revset:  "all()",
		files:   []string{},
		matches: fuzzy.Matches{},
		styles:  fuzzy_search.NewStyles(),
	}

	cmd := model.updateRevSet()
	msg := cmd()
	updateMsg, ok := msg.(common.UpdateRevSetMsg)
	assert.True(t, ok)

	// when no path is selected, should return the original revset
	assert.Equal(t, "all()", string(updateMsg))
}

func TestUpdateRevSet_EmptyMatches(t *testing.T) {
	model := &fuzzyFiles{
		revset:  "@",
		files:   []string{"file1.txt"},
		matches: fuzzy.Matches{},
		cursor:  0,
		styles:  fuzzy_search.NewStyles(),
	}

	cmd := model.updateRevSet()
	msg := cmd()
	updateMsg, ok := msg.(common.UpdateRevSetMsg)
	assert.True(t, ok)

	// when matches is empty, SelectedMatch returns empty string
	assert.Equal(t, "@", string(updateMsg))
}
