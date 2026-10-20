package jj

import "encoding/json"

// FileStat holds per-file added/removed line counts from `diff.stat()`.
type FileStat struct {
	Path    string `json:"path"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

// ParseFileStatLine parses one `json({...})` diffstat template line,
// returning false for anything else (delimiters, summary lines, legacy output).
func ParseFileStatLine(line string) (FileStat, bool) {
	var st FileStat
	if err := json.Unmarshal([]byte(line), &st); err != nil || st.Path == "" {
		return FileStat{}, false
	}
	return st, true
}
