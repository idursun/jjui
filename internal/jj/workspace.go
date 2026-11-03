package jj

import "strings"

type Workspace struct {
	Name    string
	Current bool
}

func WorkspaceList() []string {
	return []string{"workspace", "list", "--ignore-working-copy", "--color", "never", "-T", `name ++ "\t" ++ target.current_working_copy() ++ "\n"`}
}

func ParseWorkspaces(output string) []Workspace {
	var result []Workspace
	for _, line := range strings.Split(output, "\n") {
		name, current, ok := strings.Cut(line, "\t")
		if ok && name != "" {
			result = append(result, Workspace{Name: name, Current: current == "true"})
		}
	}
	return result
}
