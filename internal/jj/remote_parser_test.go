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

func TestParseRemoteListOutputFull(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []RemoteInfo
	}{
		{
			name:   "single remote",
			output: "origin https://github.com/user/repo.git\n",
			expected: []RemoteInfo{
				{Name: "origin", URL: "https://github.com/user/repo.git"},
			},
		},
		{
			name:   "multiple remotes",
			output: "origin https://github.com/user/repo.git\nupstream https://github.com/upstream/repo.git\n",
			expected: []RemoteInfo{
				{Name: "origin", URL: "https://github.com/user/repo.git"},
				{Name: "upstream", URL: "https://github.com/upstream/repo.git"},
			},
		},
		{
			name:     "empty output",
			output:   "",
			expected: nil,
		},
		{
			name:   "SSH URL",
			output: "origin git@github.com:user/repo.git\n",
			expected: []RemoteInfo{
				{Name: "origin", URL: "git@github.com:user/repo.git"},
			},
		},
		{
			name:   "mixed URLs",
			output: "origin https://github.com/user/repo.git\nupstream git@gitlab.com:upstream/repo.git\n",
			expected: []RemoteInfo{
				{Name: "origin", URL: "https://github.com/user/repo.git"},
				{Name: "upstream", URL: "git@gitlab.com:upstream/repo.git"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseRemoteListOutputFull(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}
