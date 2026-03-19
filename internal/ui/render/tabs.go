package render

import "strings"

const defaultTabWidth = 4

// ExpandTabs converts tabs to spaces while preserving ANSI escape sequences in
// the visible text. Callers should pass a single line so tab stops reset at
// newline boundaries.
func ExpandTabs(line string) string {
	if !strings.ContainsRune(line, '\t') {
		return line
	}

	var b strings.Builder
	b.Grow(len(line))

	col := 0
	rest := line
	for {
		idx := strings.IndexByte(rest, '\t')
		if idx < 0 {
			b.WriteString(rest)
			return b.String()
		}

		chunk := rest[:idx]
		b.WriteString(chunk)
		col += StringWidth(chunk)

		spaces := defaultTabWidth - (col % defaultTabWidth)
		b.WriteString(strings.Repeat(" ", spaces))
		col += spaces

		rest = rest[idx+1:]
	}
}
