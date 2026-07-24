package annotation

import "strings"

type presentation int

const (
	diffPresentation presentation = iota
	filePresentation
)

type fileItem struct {
	Path          string
	Patch         *patchFile
	Content       []string
	ContentLoaded bool
	ContentErr    error
}

type reviewDocument struct {
	revision        string
	loadingRevision string
	description     string
	files           []fileItem
	file            int
	presentation    presentation
	loading         bool
	err             error
}

func (d *reviewDocument) changeID() string {
	return d.revision
}

func (d *reviewDocument) currentFile() *fileItem {
	if d.file < 0 || d.file >= len(d.files) {
		return nil
	}
	return &d.files[d.file]
}

func (d *reviewDocument) source() reviewSource {
	return reviewSource{
		file:         d.currentFile(),
		presentation: d.presentation,
	}
}

type reviewSource struct {
	file         *fileItem
	presentation presentation
}

func (s reviewSource) Len() int {
	if s.file == nil {
		return 0
	}
	if s.presentation == filePresentation {
		return len(s.file.Content)
	}
	if s.file.Patch == nil {
		return 0
	}
	return len(s.file.Patch.Lines)
}

func (s reviewSource) Commentable(index int) bool {
	if index < 0 || index >= s.Len() {
		return false
	}
	if s.presentation == filePresentation {
		return true
	}
	return s.file.Patch.Lines[index].commentable()
}

func (s reviewSource) AnnotationLocation(start, end int) (lineRange, lineRange, string) {
	if s.file == nil || start < 0 || end < start || end >= s.Len() {
		return lineRange{}, lineRange{}, ""
	}
	if s.presentation == filePresentation {
		lines := lineRange{Start: start + 1, End: end + 1}
		return lineRange{}, lines, diffContextSnippet(s.file.Content[start : end+1])
	}

	var oldLines lineRange
	var newLines lineRange
	var snippet []string
	for _, line := range s.file.Patch.Lines[start : end+1] {
		if !line.commentable() {
			continue
		}
		snippet = append(snippet, line.Raw)
		extendRange(&oldLines, line.OldLine)
		extendRange(&newLines, line.NewLine)
	}
	return oldLines, newLines, strings.Join(snippet, "\n")
}

func (s reviewSource) Contains(index int, annotation Annotation) bool {
	if index < 0 || index >= s.Len() {
		return false
	}
	if s.presentation == filePresentation {
		return lineInRange(index+1, annotation.NewLines)
	}
	line := s.file.Patch.Lines[index]
	return lineInRange(line.NewLine, annotation.NewLines) ||
		lineInRange(line.OldLine, annotation.OldLines)
}

func (s reviewSource) Anchor(annotation Annotation) (int, bool) {
	if s.presentation == filePresentation {
		if annotation.NewLines.End <= 0 || annotation.NewLines.End > s.Len() {
			return 0, false
		}
		return annotation.NewLines.End - 1, true
	}
	if s.file == nil || s.file.Patch == nil {
		return 0, false
	}
	for index, line := range s.file.Patch.Lines {
		if annotation.NewLines.End > 0 && annotation.NewLines.End == line.NewLine {
			return index, true
		}
		if annotation.NewLines.End == 0 &&
			annotation.OldLines.End > 0 &&
			annotation.OldLines.End == line.OldLine {
			return index, true
		}
	}
	return 0, false
}

func diffContextSnippet(lines []string) string {
	contextLines := make([]string, len(lines))
	for index, line := range lines {
		contextLines[index] = " " + line
	}
	return strings.Join(contextLines, "\n")
}

func extendRange(target *lineRange, line int) {
	if line == 0 {
		return
	}
	if target.Start == 0 || line < target.Start {
		target.Start = line
	}
	if line > target.End {
		target.End = line
	}
}

func lineInRange(line int, lines lineRange) bool {
	return line > 0 && lines.Start > 0 && line >= lines.Start && line <= lines.End
}

func patchFiles(patch []patchFile) []fileItem {
	result := make([]fileItem, 0, len(patch))
	seen := make(map[string]bool)
	for index := range patch {
		path := patch[index].path()
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		result = append(result, fileItem{Path: path, Patch: &patch[index]})
	}
	return result
}

func splitFileLines(value string) []string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.TrimSuffix(value, "\n")
	if value == "" {
		return []string{""}
	}
	return strings.Split(value, "\n")
}
