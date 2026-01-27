package provider

import (
	"fmt"
	"net/url"
	"strings"
)

// BuildPRURL constructs a PR/MR creation URL for the given remote and branch.
// Supports GitHub, GitLab, and Bitbucket. Returns empty string for unknown providers.
func BuildPRURL(remoteURL, sourceBranch, targetBranch string) string {
	host, owner, repo, ok := parseGitURL(remoteURL)
	if !ok || sourceBranch == "" {
		return ""
	}
	if targetBranch == "" {
		targetBranch = "main"
	}

	baseURL := "https://" + host
	hostLower := strings.ToLower(host)

	switch {
	case strings.Contains(hostLower, "github"):
		return fmt.Sprintf("%s/%s/%s/compare/%s...%s?expand=1",
			baseURL, owner, repo,
			url.PathEscape(targetBranch),
			url.PathEscape(sourceBranch))

	case strings.Contains(hostLower, "gitlab"):
		return fmt.Sprintf("%s/%s/%s/-/merge_requests/new?merge_request[source_branch]=%s&merge_request[target_branch]=%s",
			baseURL, owner, repo,
			url.QueryEscape(sourceBranch),
			url.QueryEscape(targetBranch))

	case strings.Contains(hostLower, "bitbucket"):
		return fmt.Sprintf("%s/%s/%s/pull-requests/new?source=%s&dest=%s",
			baseURL, owner, repo,
			url.QueryEscape(sourceBranch),
			url.QueryEscape(targetBranch))

	default:
		return ""
	}
}

func ProviderName(remoteURL string) string {
	host, _, _, ok := parseGitURL(remoteURL)
	if !ok {
		return ""
	}
	hostLower := strings.ToLower(host)

	switch {
	case strings.Contains(hostLower, "github"):
		return "GitHub"
	case strings.Contains(hostLower, "gitlab"):
		return "GitLab"
	case strings.Contains(hostLower, "bitbucket"):
		return "Bitbucket"
	default:
		return ""
	}
}

func parseGitURL(remoteURL string) (host, owner, repo string, ok bool) {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return "", "", "", false
	}

	// Detect SSH vs HTTPS: SSH has ":" but not "://"
	if strings.Contains(remoteURL, ":") && !strings.Contains(remoteURL, "://") {
		return parseSSHURL(remoteURL)
	}
	return parseHTTPSURL(remoteURL)
}

// parseSSHURL parses SSH-style URLs: [user@]host:owner/repo[.git]
func parseSSHURL(remoteURL string) (host, owner, repo string, ok bool) {
	parts := strings.SplitN(remoteURL, ":", 2)
	if len(parts) != 2 {
		return "", "", "", false
	}

	host = parts[0]
	if idx := strings.LastIndex(host, "@"); idx != -1 {
		host = host[idx+1:]
	}

	path := strings.TrimSuffix(parts[1], ".git")
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 2 {
		return "", "", "", false
	}

	return host, pathParts[0], pathParts[1], true
}

// parseHTTPSURL parses HTTPS-style URLs: https://host/owner/repo[.git]
func parseHTTPSURL(remoteURL string) (host, owner, repo string, ok bool) {
	parsed, err := url.Parse(remoteURL)
	if err != nil {
		return "", "", "", false
	}

	host = parsed.Host
	path := strings.TrimPrefix(parsed.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		return "", "", "", false
	}

	return host, parts[0], parts[1], true
}
