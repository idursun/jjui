package jj

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRemoteListOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []string
	}{
		{
			name:     "single remote",
			output:   "origin https://github.com/user/repo.git\n",
			expected: []string{"origin"},
		},
		{
			name:     "multiple remotes",
			output:   "origin https://github.com/user/repo.git\nupstream https://github.com/upstream/repo.git\nfork https://github.com/fork/repo.git\n",
			expected: []string{"origin", "upstream", "fork"},
		},
		{
			name:     "empty output",
			output:   "",
			expected: []string{},
		},
		{
			name:     "with trailing newline",
			output:   "origin https://github.com/user/repo.git\nupstream https://github.com/upstream/repo.git\n",
			expected: []string{"origin", "upstream"},
		},
		{
			name:     "with extra spaces",
			output:   "  origin   https://github.com/user/repo.git  \n  upstream   https://github.com/upstream/repo.git  \n",
			expected: []string{"origin", "upstream"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseRemoteListOutput(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}
