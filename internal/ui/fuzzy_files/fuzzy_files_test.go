package fuzzy_files

import (
	"testing"

	"github.com/idursun/jjui/internal/ui/common"
	"github.com/sahilm/fuzzy"
	"github.com/stretchr/testify/assert"
)

func TestUpdateRevSet_WithPath(t *testing.T) {
	model := &fuzzyFiles{
		revset: "all()",
		paths:  []string{"file1.txt", "path/to/file2.go", "special file.txt"},
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
		paths:  []string{"file with spaces.txt"},
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
		paths:  []string{"file{with}braces.txt"},
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

func TestUpdateRevSet_WithDirectory(t *testing.T) {
	model := &fuzzyFiles{
		revset: "all()",
		paths:  []string{"path/to/"},
	}

	model.matches = fuzzy.Matches{{Index: 0, Str: "path/to/"}}
	model.cursor = 0

	cmd := model.updateRevSet()
	msg := cmd()
	updateMsg, ok := msg.(common.UpdateRevSetMsg)
	assert.True(t, ok)
	assert.Equal(t, "files('path/to/')", string(updateMsg))
}

func TestUpdateRevSet_NoPath(t *testing.T) {
	model := &fuzzyFiles{
		revset:  "all()",
		paths:   []string{},
		matches: fuzzy.Matches{},
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
		paths:   []string{"file1.txt"},
		matches: fuzzy.Matches{},
		cursor:  0,
	}

	cmd := model.updateRevSet()
	msg := cmd()
	updateMsg, ok := msg.(common.UpdateRevSetMsg)
	assert.True(t, ok)

	// when matches is empty, SelectedMatch returns empty string
	assert.Equal(t, "@", string(updateMsg))
}

func TestBuildPathEntries_IncludesDirectories(t *testing.T) {
	entries := buildPathEntries([]byte("src/pkg/main.go\nsrc/other.go\nREADME.md\n"))

	assert.Equal(t, []string{
		"src/",
		"src/pkg/",
		"src/pkg/main.go",
		"src/other.go",
		"README.md",
	}, entries)
}

func TestBuildPathEntries_EmptyOutput(t *testing.T) {
	assert.Empty(t, buildPathEntries(nil))
}
