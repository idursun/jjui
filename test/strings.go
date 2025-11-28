package test

import "strings"

// Stripped normalizes a string for comparisons by trimming surrounding
// whitespace, trimming each individual line, and removing carriage returns.
func Stripped(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.TrimSpace(s)

	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}

	return strings.Join(lines, "\n")
}
