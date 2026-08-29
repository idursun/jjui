package jj

import (
	"path"
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type SummaryFile struct {
	Status      rune
	FileName    FileName
	RenameFrom  FileName
	braceRename bool
}

var braceRenameRe = regexp.MustCompile(`^(.*)\{([^{}]*?) =>\s*([^{}]*?)\}(.*)$`)

func ParseSummaryFile(line string) (SummaryFile, bool) {
	line = strings.TrimSpace(ansi.Strip(line))
	if line == "" {
		return SummaryFile{}, false
	}

	var status rune
	if fields := strings.Fields(line); len(fields) > 1 && isSummaryStatus(fields[0]) {
		status = []rune(fields[0])[0]
		line = strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
	}

	result := SummaryFile{Status: status, FileName: NewFileName(line)}
	if matches := braceRenameRe.FindStringSubmatch(line); matches != nil {
		result.RenameFrom = NewFileName(path.Clean(matches[1] + strings.TrimSpace(matches[2]) + matches[4]))
		result.FileName = NewFileName(path.Clean(matches[1] + strings.TrimSpace(matches[3]) + matches[4]))
		result.braceRename = true
		return result, true
	}
	if before, after, ok := strings.Cut(line, " => "); ok {
		result.RenameFrom = NewFileName(strings.TrimSpace(before))
		result.FileName = NewFileName(strings.TrimSpace(after))
	}
	return result, true
}

func (s SummaryFile) Display(repoRoot, workingDirectory string) string {
	to := s.FileName.Display(repoRoot, workingDirectory)
	if s.RenameFrom.IsEmpty() {
		return to
	}
	from := s.RenameFrom.Display(repoRoot, workingDirectory)
	if !s.braceRename {
		return from + " => " + to
	}
	return compactRename(from, to)
}

func compactRename(from, to string) string {
	a, b := []rune(from), []rune(to)
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	aEnd, bEnd := len(a)-suffix, len(b)-suffix
	return string(a[:prefix]) + "{" + string(a[prefix:aEnd]) + " => " + string(b[prefix:bEnd]) + "}" + string(a[aEnd:])
}

func isSummaryStatus(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		if !strings.ContainsRune("ACDMR?!", r) {
			return false
		}
	}
	return true
}
