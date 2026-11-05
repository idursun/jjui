package preview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/theme"
)

// parseFileStats builds a per-file stat lookup from `jj.FileStats` output.
func parseFileStats(output []byte) map[string]jj.FileStat {
	stats := make(map[string]jj.FileStat)
	for line := range strings.Lines(string(output)) {
		if st, ok := jj.ParseFileStatLine(strings.TrimSpace(line)); ok {
			stats[st.Path] = st
		}
	}
	return stats
}

// annotateFileStats appends +added/-removed counts to preview lines that name
// changed files, so revision previews show per-file numbers without opening
// the details. It recognizes bare path lines, `jj show`/`jj diff` file
// headers ("<path> --- <language>"), and summary lines ("M <path>"). Both
// repository-relative and display-relative (cwd-relative) paths match.
func annotateFileStats(content string, stats map[string]jj.FileStat, display func(string) string) string {
	if len(stats) == 0 {
		return content
	}
	lookup := make(map[string]jj.FileStat, len(stats)*2)
	var headers []string
	for path, st := range stats {
		lookup[path] = st
		headers = append(headers, path)
		if alt := display(path); alt != "" && alt != path {
			lookup[alt] = st
			headers = append(headers, alt)
		}
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(ansi.Strip(line))
		if trimmed == "" {
			continue
		}
		st, ok := lookup[trimmed]
		if !ok && strings.Contains(trimmed, " --- ") {
			for _, h := range headers {
				if strings.HasPrefix(trimmed, h+" --- ") {
					st, ok = lookup[h]
					break
				}
			}
		}
		if !ok {
			if summary, valid := jj.ParseSummaryFile(trimmed); valid && !summary.FileName.IsEmpty() {
				st, ok = stats[summary.FileName.Path()]
			}
		}
		if !ok || st.Added+st.Removed == 0 {
			continue
		}
		lines[i] = line + diffstatSuffix(st)
	}
	return strings.Join(lines, "\n")
}

func diffstatSuffix(st jj.FileStat) string {
	var sb strings.Builder
	if st.Added > 0 {
		sb.WriteString(" ")
		sb.WriteString(theme.DefaultPalette.Get("preview", "", "added", false).Render(fmt.Sprintf("+%d", st.Added)))
	}
	if st.Removed > 0 {
		sb.WriteString(" ")
		sb.WriteString(theme.DefaultPalette.Get("preview", "", "deleted", false).Render(fmt.Sprintf("-%d", st.Removed)))
	}
	return sb.String()
}
