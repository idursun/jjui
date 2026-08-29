package fuzzy_files

import (
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/sahilm/fuzzy"
	"github.com/stretchr/testify/assert"
)

func TestUpdateRevSet_WithPath(t *testing.T) {
	model := &fuzzyFiles{
		revset: "all()",
		paths:  fileNames("file1.txt", "path/to/file2.go", "special file.txt"),
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

	assert.Equal(t, `files(file:"path/to/file2.go")`, string(updateMsg))
}

func TestUpdateRevSet_WithPathContainingSpaces(t *testing.T) {
	model := &fuzzyFiles{
		revset: "all()",
		paths:  fileNames("file with spaces.txt"),
	}

	model.matches = fuzzy.Matches{
		{Index: 0, Str: "file with spaces.txt"},
	}
	model.cursor = 0

	cmd := model.updateRevSet()
	msg := cmd()
	updateMsg, ok := msg.(common.UpdateRevSetMsg)
	assert.True(t, ok)

	assert.Equal(t, `files(file:"file with spaces.txt")`, string(updateMsg))
}

func TestUpdateRevSet_WithPathContainingBraces(t *testing.T) {
	model := &fuzzyFiles{
		revset: "all()",
		paths:  fileNames("file{with}braces.txt"),
	}

	model.matches = fuzzy.Matches{
		{Index: 0, Str: "file{with}braces.txt"},
	}
	model.cursor = 0

	cmd := model.updateRevSet()
	msg := cmd()
	updateMsg, ok := msg.(common.UpdateRevSetMsg)
	assert.True(t, ok)

	assert.Equal(t, `files(file:"file{with}braces.txt")`, string(updateMsg))
}

func TestUpdateRevSet_WithDirectory(t *testing.T) {
	model := &fuzzyFiles{
		revset: "all()",
		paths:  fileNames("path/to/"),
	}

	model.matches = fuzzy.Matches{{Index: 0, Str: "path/to/"}}
	model.cursor = 0

	cmd := model.updateRevSet()
	msg := cmd()
	updateMsg, ok := msg.(common.UpdateRevSetMsg)
	assert.True(t, ok)
	assert.Equal(t, `files(file:"path/to/")`, string(updateMsg))
}

func TestUpdateRevSet_NoPath(t *testing.T) {
	model := &fuzzyFiles{
		revset:  "all()",
		paths:   []jj.FileName{},
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
		paths:   fileNames("file1.txt"),
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

	assert.Equal(t, fileNames(
		"src/",
		"src/pkg/",
		"src/pkg/main.go",
		"src/other.go",
		"README.md",
	), entries)
}

func TestFileSearchDisplaysRelativePathsButUsesRepositoryPath(t *testing.T) {
	model := &fuzzyFiles{
		revset:           "all()",
		paths:            fileNames("internal/ui/ui.go"),
		repoRoot:         "/work/repo",
		workingDirectory: "/work/repo/internal",
		matches:          fuzzy.Matches{{Index: 0, Str: "ui/ui.go"}},
	}

	assert.Equal(t, "ui/ui.go", model.String(0))
	msg := model.updateRevSet()()
	assert.Equal(t, `files(file:"internal/ui/ui.go")`, string(msg.(common.UpdateRevSetMsg)))

	edit := model.handleIntent(intents.FileSearchEdit{})().(common.ExecMsg)
	assert.Contains(t, edit.Line, " 'internal/ui/ui.go'")
}

func TestFileSearchSafelyEscapesApostrophes(t *testing.T) {
	model := &fuzzyFiles{
		revset:  "all()",
		paths:   fileNames("it's complicated.go"),
		matches: fuzzy.Matches{{Index: 0, Str: "it's complicated.go"}},
	}

	msg := model.updateRevSet()()
	assert.Equal(t, `files(file:"it's complicated.go")`, string(msg.(common.UpdateRevSetMsg)))

	edit := model.handleIntent(intents.FileSearchEdit{})().(common.ExecMsg)
	assert.Contains(t, edit.Line, ` 'it'\''s complicated.go'`)
}

func fileNames(paths ...string) []jj.FileName {
	result := make([]jj.FileName, len(paths))
	for i, path := range paths {
		result[i] = jj.NewFileName(path)
	}
	return result
}

func TestBuildPathEntries_EmptyOutput(t *testing.T) {
	assert.Empty(t, buildPathEntries(nil))
}
