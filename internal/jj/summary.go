package jj

import (
	"path"
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type SummaryFile struct {
	Status   rune
	Name     string
	FileName string
}

var braceRenameRe = regexp.MustCompile(`\{[^}]*? => \s*([^}]*?)\s*\}`)

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

	return SummaryFile{
		Status:   status,
		Name:     line,
		FileName: NormalizeSummaryFileName(line),
	}, true
}

func NormalizeSummaryFileName(fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return ""
	}

	if strings.Contains(fileName, "{") {
		return path.Clean(braceRenameRe.ReplaceAllString(fileName, "$1"))
	}
	if _, after, ok := strings.Cut(fileName, " => "); ok {
		return strings.TrimSpace(after)
	}
	return fileName
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
