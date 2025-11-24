package oplog

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expected int
	}{
		{
			name:     "single",
			fileName: "single.log",
			expected: 1,
		},
		{
			name:     "multi",
			fileName: "multi.log",
			expected: 2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file, err := os.Open(fmt.Sprintf("testdata/%s", test.fileName))
			assert.NoError(t, err)

			rows := parseRows(file)
			assert.Len(t, rows, test.expected)
		})
	}
}
