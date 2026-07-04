package jj

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSummaryFile(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		status   rune
		display  string
		fileName string
	}{
		{
			name:     "normal status",
			line:     "M src/main.go",
			status:   'M',
			display:  "src/main.go",
			fileName: "src/main.go",
		},
		{
			name:     "colored status",
			line:     "\x1b[31mA docs/readme.md\x1b[0m",
			status:   'A',
			display:  "docs/readme.md",
			fileName: "docs/readme.md",
		},
		{
			name:     "brace rename",
			line:     "R internal/ui/{revisions => }/file.go",
			status:   'R',
			display:  "internal/ui/{revisions => }/file.go",
			fileName: "internal/ui/file.go",
		},
		{
			name:     "deep brace rename",
			line:     "R {src1/to_be_renamed.md => src2/renamed.md}",
			status:   'R',
			display:  "{src1/to_be_renamed.md => src2/renamed.md}",
			fileName: "src2/renamed.md",
		},
		{
			name:     "plain rename",
			line:     "old/path.go => new/path.go",
			display:  "old/path.go => new/path.go",
			fileName: "new/path.go",
		},
		{
			name:     "braces without rename",
			line:     "M file{with}braces.txt",
			status:   'M',
			display:  "file{with}braces.txt",
			fileName: "file{with}braces.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseSummaryFile(tt.line)
			require.True(t, ok)
			assert.Equal(t, tt.status, got.Status)
			assert.Equal(t, tt.display, got.Name)
			assert.Equal(t, tt.fileName, got.FileName)
		})
	}
}
