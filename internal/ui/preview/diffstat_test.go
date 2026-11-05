package preview

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func testStats() map[string]jj.FileStat {
	return map[string]jj.FileStat{
		"internal/config/config.go": {Path: "internal/config/config.go", Added: 12, Removed: 3},
		"docs/preview.mdx":          {Path: "docs/preview.mdx", Added: 5, Removed: 0},
		"bin/blob.dat":              {Path: "bin/blob.dat", Added: 0, Removed: 0},
	}
}

func identityDisplay(path string) string { return path }

func TestAnnotateFileStats(t *testing.T) {
	content := "internal/config/config.go\ndocs/preview.mdx\nunrelated line\n"
	got := test.Stripped(annotateFileStats(content, testStats(), identityDisplay))

	assert.Contains(t, got, "internal/config/config.go +12 -3")
	assert.Contains(t, got, "docs/preview.mdx +5")
	assert.Contains(t, got, "unrelated line")
	assert.NotContains(t, got, "unrelated line +")
}

func TestAnnotateFileStatsShowHeaders(t *testing.T) {
	// `jj show`/`jj diff` file header, plain and with ANSI colors.
	content := "internal/config/config.go --- Go\n" +
		"\x1b[1m\x1b[93mdocs/preview.mdx\x1b[39m\x1b[0m\x1b[2m --- XML\x1b[0m\n"
	got := test.Stripped(ansi.Strip(annotateFileStats(content, testStats(), identityDisplay)))

	assert.Contains(t, got, "internal/config/config.go --- Go +12 -3")
	assert.Contains(t, got, "docs/preview.mdx --- XML +5")
}

func TestAnnotateFileStatsSummaryLines(t *testing.T) {
	content := "M internal/config/config.go\nA docs/preview.mdx\nM untouched.go\n"
	got := test.Stripped(annotateFileStats(content, testStats(), identityDisplay))

	assert.Contains(t, got, "M internal/config/config.go +12 -3")
	assert.Contains(t, got, "A docs/preview.mdx +5")
	assert.NotContains(t, got, "untouched.go +")
}

func TestAnnotateFileStatsSkipsZeroCounts(t *testing.T) {
	got := test.Stripped(annotateFileStats("bin/blob.dat\n", testStats(), identityDisplay))
	assert.Equal(t, "bin/blob.dat", got)
}

func TestAnnotateFileStatsMatchesDisplayPaths(t *testing.T) {
	display := func(path string) string {
		if path == "internal/config/config.go" {
			return "config.go"
		}
		return path
	}
	got := test.Stripped(annotateFileStats("config.go\n", testStats(), display))
	assert.Contains(t, got, "config.go +12 -3")
}

func TestAnnotateFileStatsEmpty(t *testing.T) {
	assert.Equal(t, "nothing\n", annotateFileStats("nothing\n", nil, identityDisplay))
	assert.Equal(t, "nothing\n", annotateFileStats("nothing\n", map[string]jj.FileStat{}, identityDisplay))
}

func TestParseFileStats(t *testing.T) {
	stats := parseFileStats([]byte("{\"path\":\"a.go\",\"added\":1,\"removed\":2}\nnot a stat line\n{\"path\":\"b.go\",\"added\":0,\"removed\":0}\n"))
	assert.Equal(t, map[string]jj.FileStat{
		"a.go": {Path: "a.go", Added: 1, Removed: 2},
		"b.go": {Path: "b.go", Added: 0, Removed: 0},
	}, stats)
}
