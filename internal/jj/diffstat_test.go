package jj

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFileStatLine(t *testing.T) {
	st, ok := ParseFileStatLine(`{"path":"a/b.go","added":12,"removed":3}`)
	require.True(t, ok)
	assert.Equal(t, FileStat{Path: "a/b.go", Added: 12, Removed: 3}, st)
}

func TestParseFileStatLineRejectsNonStatLines(t *testing.T) {
	for _, line := range []string{
		"",
		"$$",
		"M a/b.go",
		`{"added":1}`,
		`not json`,
	} {
		_, ok := ParseFileStatLine(line)
		assert.False(t, ok, "line %q must not parse as a stat line", line)
	}
}
