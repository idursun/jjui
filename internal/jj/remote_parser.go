package jj

import (
	"slices"
	"strings"

	"github.com/idursun/jjui/internal/config"
)

type RemoteInfo struct {
	Name string
	URL  string
}

func ParseRemoteListOutput(output string) []string {
	defaultRemote := config.GetGitDefaultRemote(config.Current)
	remotes := []string{}
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			remotes = append(remotes, strings.Fields(name)[0])
		}
	}
	// Move defaultRemote to front if present
	if i := slices.Index(remotes, defaultRemote); i >= 0 {
		remotes = append([]string{defaultRemote}, append(remotes[:i], remotes[i+1:]...)...)
	}
	return remotes
}

// ParseRemoteListOutputFull parses `jj git remote list` output and returns
// both name and URL for each remote. The output format is:
// remote_name URL
func ParseRemoteListOutputFull(output string) []RemoteInfo {
	defaultRemote := config.GetGitDefaultRemote(config.Current)
	var remotes []RemoteInfo
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			remotes = append(remotes, RemoteInfo{
				Name: fields[0],
				URL:  fields[1],
			})
		} else if len(fields) == 1 {
			// Handle case where only name is present (shouldn't happen but be safe)
			remotes = append(remotes, RemoteInfo{
				Name: fields[0],
				URL:  "",
			})
		}
	}
	// Move defaultRemote to front if present
	idx := slices.IndexFunc(remotes, func(r RemoteInfo) bool {
		return r.Name == defaultRemote
	})
	if idx > 0 {
		defaultInfo := remotes[idx]
		remotes = append([]RemoteInfo{defaultInfo}, append(remotes[:idx], remotes[idx+1:]...)...)
	}
	return remotes
}
