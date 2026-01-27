package provider

import (
	"testing"
)

func TestBuildPRURL(t *testing.T) {
	tests := []struct {
		name         string
		remoteURL    string
		sourceBranch string
		targetBranch string
		expected     string
	}{
		{
			name:         "GitHub SSH",
			remoteURL:    "git@github.com:owner/repo.git",
			sourceBranch: "feature",
			targetBranch: "main",
			expected:     "https://github.com/owner/repo/compare/main...feature?expand=1",
		},
		{
			name:         "GitHub HTTPS",
			remoteURL:    "https://github.com/owner/repo.git",
			sourceBranch: "feature",
			targetBranch: "main",
			expected:     "https://github.com/owner/repo/compare/main...feature?expand=1",
		},
		{
			name:         "GitHub default target branch",
			remoteURL:    "git@github.com:owner/repo.git",
			sourceBranch: "feature",
			targetBranch: "",
			expected:     "https://github.com/owner/repo/compare/main...feature?expand=1",
		},
		{
			name:         "GitLab SSH",
			remoteURL:    "git@gitlab.com:owner/repo.git",
			sourceBranch: "feature",
			targetBranch: "main",
			expected:     "https://gitlab.com/owner/repo/-/merge_requests/new?merge_request[source_branch]=feature&merge_request[target_branch]=main",
		},
		{
			name:         "GitLab HTTPS",
			remoteURL:    "https://gitlab.com/owner/repo.git",
			sourceBranch: "feature",
			targetBranch: "main",
			expected:     "https://gitlab.com/owner/repo/-/merge_requests/new?merge_request[source_branch]=feature&merge_request[target_branch]=main",
		},
		{
			name:         "Bitbucket SSH",
			remoteURL:    "git@bitbucket.org:owner/repo.git",
			sourceBranch: "feature",
			targetBranch: "main",
			expected:     "https://bitbucket.org/owner/repo/pull-requests/new?source=feature&dest=main",
		},
		{
			name:         "Bitbucket HTTPS",
			remoteURL:    "https://bitbucket.org/owner/repo.git",
			sourceBranch: "feature",
			targetBranch: "main",
			expected:     "https://bitbucket.org/owner/repo/pull-requests/new?source=feature&dest=main",
		},
		{
			name:         "Branch with slash",
			remoteURL:    "git@github.com:owner/repo.git",
			sourceBranch: "feature/my-feature",
			targetBranch: "main",
			expected:     "https://github.com/owner/repo/compare/main...feature%2Fmy-feature?expand=1",
		},
		{
			name:         "Unknown provider",
			remoteURL:    "git@example.com:owner/repo.git",
			sourceBranch: "feature",
			targetBranch: "main",
			expected:     "",
		},
		{
			name:         "Empty remote URL",
			remoteURL:    "",
			sourceBranch: "feature",
			targetBranch: "main",
			expected:     "",
		},
		{
			name:         "Empty source branch",
			remoteURL:    "git@github.com:owner/repo.git",
			sourceBranch: "",
			targetBranch: "main",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPRURL(tt.remoteURL, tt.sourceBranch, tt.targetBranch)
			if result != tt.expected {
				t.Errorf("BuildPRURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestProviderName(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		expected  string
	}{
		{"GitHub SSH", "git@github.com:owner/repo.git", "GitHub"},
		{"GitHub HTTPS", "https://github.com/owner/repo.git", "GitHub"},
		{"GitHub Enterprise", "git@github.mycompany.com:owner/repo.git", "GitHub"},
		{"GitLab SSH", "git@gitlab.com:owner/repo.git", "GitLab"},
		{"GitLab HTTPS", "https://gitlab.com/owner/repo.git", "GitLab"},
		{"Bitbucket SSH", "git@bitbucket.org:owner/repo.git", "Bitbucket"},
		{"Bitbucket HTTPS", "https://bitbucket.org/owner/repo.git", "Bitbucket"},
		{"Unknown", "git@example.com:owner/repo.git", ""},
		{"Empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProviderName(tt.remoteURL)
			if result != tt.expected {
				t.Errorf("ProviderName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseGitURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedHost  string
		expectedOwner string
		expectedRepo  string
		expectOk      bool
	}{
		{"GitHub SSH", "git@github.com:owner/repo.git", "github.com", "owner", "repo", true},
		{"GitHub SSH no .git", "git@github.com:owner/repo", "github.com", "owner", "repo", true},
		{"GitHub HTTPS", "https://github.com/owner/repo.git", "github.com", "owner", "repo", true},
		{"GitHub HTTPS no .git", "https://github.com/owner/repo", "github.com", "owner", "repo", true},
		{"With dashes", "git@github.com:my-org/my-repo.git", "github.com", "my-org", "my-repo", true},
		{"Empty URL", "", "", "", "", false},
		{"Invalid URL", "not-a-url", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, owner, repo, ok := parseGitURL(tt.url)
			if ok != tt.expectOk {
				t.Errorf("parseGitURL() ok = %v, want %v", ok, tt.expectOk)
			}
			if host != tt.expectedHost {
				t.Errorf("parseGitURL() host = %q, want %q", host, tt.expectedHost)
			}
			if owner != tt.expectedOwner {
				t.Errorf("parseGitURL() owner = %q, want %q", owner, tt.expectedOwner)
			}
			if repo != tt.expectedRepo {
				t.Errorf("parseGitURL() repo = %q, want %q", repo, tt.expectedRepo)
			}
		})
	}
}
