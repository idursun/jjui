package preview

import (
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// wrapContent word-wraps each line of content to fit within width cells.
// Continuation lines keep the leading indentation of the line they belong to
// (e.g. the indented description in `jj show`), and ANSI styles and hyperlinks
// are re-applied on every line so each line renders correctly on its own,
// even when it is the first one visible in the viewport.
func wrapContent(content string, width int) string {
	if width <= 0 {
		return content
	}
	var lines []string
	for line := range strings.SplitSeq(content, "\n") {
		lines = append(lines, wrapLine(line, width)...)
	}

	var sb strings.Builder
	w := lipgloss.NewWrapWriter(&sb)
	_, _ = io.WriteString(w, strings.Join(lines, "\n"))
	_ = w.Close()
	return sb.String()
}

func wrapLine(line string, width int) []string {
	if ansi.StringWidth(line) <= width {
		return []string{line}
	}

	plain := ansi.Strip(line)
	indent := len(plain) - len(strings.TrimLeft(plain, " "))
	// Hanging indentation is only worth it when there is room left for text.
	if indent*2 > width {
		indent = 0
	}

	head, rest := "", line
	if indent > 0 {
		head = ansi.Truncate(line, indent, "")
		rest = ansi.TruncateLeft(line, indent, "")
	}

	wrapped := strings.Split(ansi.Wrap(rest, width-indent, ""), "\n")
	padding := strings.Repeat(" ", indent)
	for i := range wrapped {
		if i == 0 {
			wrapped[i] = head + wrapped[i]
		} else {
			wrapped[i] = padding + wrapped[i]
		}
	}
	return wrapped
}
