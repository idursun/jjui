package annotation

import (
	"strconv"
	"strings"

	"github.com/mattn/go-shellwords"
)

type lineKind int

const (
	lineMetadata lineKind = iota
	lineHunk
	lineContext
	lineAdded
	lineRemoved
)

type patchLine struct {
	Kind        lineKind
	Raw         string
	Content     string
	OldLine     int
	NewLine     int
	PairContent string
}

func (l patchLine) commentable() bool {
	return l.Kind == lineContext || l.Kind == lineAdded || l.Kind == lineRemoved
}

type patchFile struct {
	OldPath string
	NewPath string
	Lines   []patchLine
}

func (f patchFile) path() string {
	if f.NewPath != "" {
		return f.NewPath
	}
	return f.OldPath
}

func parseGitPatch(content string) []patchFile {
	var files []patchFile
	var current *patchFile
	oldLine := 0
	newLine := 0
	inHunk := false

	for raw := range strings.SplitSeq(strings.ReplaceAll(content, "\r", ""), "\n") {
		if strings.HasPrefix(raw, "diff --git ") {
			oldPath, newPath := parseDiffHeader(raw)
			files = append(files, patchFile{OldPath: oldPath, NewPath: newPath})
			current = &files[len(files)-1]
			if oldPath != "" && newPath != "" && oldPath != newPath {
				summary := oldPath + " -> " + newPath
				current.Lines = append(current.Lines, patchLine{
					Kind: lineMetadata, Raw: summary, Content: summary,
				})
			}
			inHunk = false
			continue
		}
		if current == nil {
			continue
		}
		if strings.HasPrefix(raw, "--- ") && !inHunk {
			current.OldPath = parseMarkerPath(strings.TrimPrefix(raw, "--- "), "a/")
			continue
		}
		if strings.HasPrefix(raw, "+++ ") && !inHunk {
			current.NewPath = parseMarkerPath(strings.TrimPrefix(raw, "+++ "), "b/")
			continue
		}
		if oldStart, newStart, ok := parseHunkHeader(raw); ok {
			oldLine = oldStart
			newLine = newStart
			inHunk = true
			current.Lines = append(current.Lines, patchLine{Kind: lineHunk, Raw: raw, Content: raw})
			continue
		}
		if !inHunk {
			if shouldShowMetadata(raw) {
				current.Lines = append(current.Lines, patchLine{Kind: lineMetadata, Raw: raw, Content: raw})
			}
			continue
		}
		if raw == `\ No newline at end of file` {
			current.Lines = append(current.Lines, patchLine{Kind: lineMetadata, Raw: raw, Content: raw})
			continue
		}
		if raw == "" {
			continue
		}
		switch raw[0] {
		case ' ':
			current.Lines = append(current.Lines, patchLine{
				Kind: lineContext, Raw: raw, Content: raw[1:], OldLine: oldLine, NewLine: newLine,
			})
			oldLine++
			newLine++
		case '-':
			current.Lines = append(current.Lines, patchLine{
				Kind: lineRemoved, Raw: raw, Content: raw[1:], OldLine: oldLine,
			})
			oldLine++
		case '+':
			current.Lines = append(current.Lines, patchLine{
				Kind: lineAdded, Raw: raw, Content: raw[1:], NewLine: newLine,
			})
			newLine++
		default:
			inHunk = false
			if shouldShowMetadata(raw) {
				current.Lines = append(current.Lines, patchLine{Kind: lineMetadata, Raw: raw, Content: raw})
			}
		}
	}

	for i := range files {
		pairChangedBlocks(files[i].Lines)
	}
	return files
}

func parseDiffHeader(line string) (string, string) {
	fields, err := shellwords.Parse(strings.TrimPrefix(line, "diff --git "))
	if err != nil || len(fields) < 2 {
		return "", ""
	}
	return trimPatchPrefix(fields[0], "a/"), trimPatchPrefix(fields[1], "b/")
}

func parseMarkerPath(path, prefix string) string {
	if path == "/dev/null" {
		return ""
	}
	fields, err := shellwords.Parse(path)
	if err == nil && len(fields) > 0 {
		path = fields[0]
	}
	return trimPatchPrefix(path, prefix)
}

func trimPatchPrefix(path, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}

func shouldShowMetadata(line string) bool {
	if line == "" ||
		strings.HasPrefix(line, "index ") ||
		strings.HasPrefix(line, "rename from ") ||
		strings.HasPrefix(line, "rename to ") {
		return false
	}
	return true
}

func parseHunkHeader(line string) (int, int, bool) {
	if !strings.HasPrefix(line, "@@") {
		return 0, 0, false
	}
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return 0, 0, false
	}
	oldStart, ok := parseRangeStart(fields[1], '-')
	if !ok {
		return 0, 0, false
	}
	newStart, ok := parseRangeStart(fields[2], '+')
	if !ok {
		return 0, 0, false
	}
	return oldStart, newStart, true
}

func parseRangeStart(field string, prefix byte) (int, bool) {
	if field == "" || field[0] != prefix {
		return 0, false
	}
	value := field[1:]
	if before, _, ok := strings.Cut(value, ","); ok {
		value = before
	}
	n, err := strconv.Atoi(value)
	return n, err == nil
}

func pairChangedBlocks(lines []patchLine) {
	for i := 0; i < len(lines); {
		if lines[i].Kind != lineRemoved {
			i++
			continue
		}
		removedStart := i
		for i < len(lines) && lines[i].Kind == lineRemoved {
			i++
		}
		addedStart := i
		for i < len(lines) && lines[i].Kind == lineAdded {
			i++
		}
		for offset := 0; removedStart+offset < addedStart && addedStart+offset < i; offset++ {
			removed := removedStart + offset
			added := addedStart + offset
			lines[removed].PairContent = lines[added].Content
			lines[added].PairContent = lines[removed].Content
		}
	}
}
