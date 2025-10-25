package jj

import (
	"slices"
	"strings"

	"github.com/idursun/jjui/internal/config"
)

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
