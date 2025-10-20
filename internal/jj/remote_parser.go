package jj

import "strings"

func ParseRemoteListOutput(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	remotes := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				remotes = append(remotes, parts[0])
			}
		}
	}
	// TODO: handle cases where there's no remote set up
	return remotes
}
