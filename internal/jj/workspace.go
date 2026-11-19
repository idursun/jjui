package jj

import (
	"encoding/json"
	"strings"
)

type Workspace struct {
	Name    string
	Current bool
	Path    string
}

func WorkspaceList() []string {
	return []string{"workspace", "list", "--ignore-working-copy", "--color", "never", "-T", `name ++ "\t" ++ target.current_working_copy() ++ "\n"`}
}

// WorkspaceListWithPaths requires jj 0.40. JSON escaping keeps tabs, newlines,
// and backslashes in names and paths separate from the record delimiters.
func WorkspaceListWithPaths() []string {
	return []string{"workspace", "list", "--ignore-working-copy", "--color", "never", "-T", `json(name) ++ "\t" ++ target.current_working_copy() ++ "\t" ++ json(root) ++ "\n"`}
}

// DiscoverWorkspaces falls back to the template supported by jj 0.37 when
// recorded roots are unavailable. Missing paths must be supplied by the caller.
func DiscoverWorkspaces(run func([]string) ([]byte, error)) ([]Workspace, error) {
	output, err := run(WorkspaceListWithPaths())
	if err != nil {
		output, err = run(WorkspaceList())
	}
	if err != nil {
		return nil, err
	}
	return ParseWorkspaces(string(output)), nil
}

func ParseWorkspaces(output string) []Workspace {
	var result []Workspace
	for _, line := range strings.Split(output, "\n") {
		if fields := strings.Split(line, "\t"); len(fields) == 3 {
			var name string
			var path *string
			if json.Unmarshal([]byte(fields[0]), &name) == nil && json.Unmarshal([]byte(fields[2]), &path) == nil && name != "" {
				workspace := Workspace{Name: name, Current: fields[1] == "true"}
				if path != nil {
					workspace.Path = *path
				}
				result = append(result, workspace)
			}
			continue
		}
		name, current, ok := strings.Cut(line, "\t")
		if ok && name != "" {
			result = append(result, Workspace{Name: name, Current: current == "true"})
		}
	}
	return result
}
