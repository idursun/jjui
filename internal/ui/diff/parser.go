package diff

import (
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

// Parse parses a unified diff string using go-gitdiff library
// and computes word-level diffs for all hunks
func Parse(diffText string) *ParsedDiff {
	files, _, err := gitdiff.Parse(strings.NewReader(diffText))
	if err != nil {
		// Fallback to empty result (matches current behavior - never errors)
		return &ParsedDiff{Files: make([]*DiffFile, 0)}
	}

	result := &ParsedDiff{
		Files: convertFiles(files),
	}

	// Compute word-level diffs for better highlighting
	ComputeWordDiffForDiff(result)

	return result
}

func convertFiles(gitFiles []*gitdiff.File) []*DiffFile {
	result := make([]*DiffFile, 0, len(gitFiles))
	for _, gf := range gitFiles {
		result = append(result, convertFile(gf))
	}
	return result
}

func convertFile(gf *gitdiff.File) *DiffFile {
	df := &DiffFile{
		OldPath:  gf.OldName,
		NewPath:  gf.NewName,
		IsBinary: gf.IsBinary,
		Status:   determineFileStatus(gf),
		Hunks:    convertFragments(gf.TextFragments),
	}
	return df
}

func determineFileStatus(gf *gitdiff.File) FileStatus {
	if gf.IsNew {
		return FileAdded
	}
	if gf.IsDelete {
		return FileDeleted
	}
	if gf.IsRename {
		return FileRenamed
	}
	if gf.IsCopy {
		return FileCopied
	}
	return FileModified
}

func convertFragments(fragments []*gitdiff.TextFragment) []Hunk {
	hunks := make([]Hunk, 0, len(fragments))
	for _, tf := range fragments {
		hunks = append(hunks, convertFragment(tf))
	}
	return hunks
}

func convertFragment(tf *gitdiff.TextFragment) Hunk {
	hunk := Hunk{
		OldStart: int(tf.OldPosition),
		OldCount: int(tf.OldLines),
		NewStart: int(tf.NewPosition),
		NewCount: int(tf.NewLines),
		Header:   strings.TrimSpace(tf.Comment),
		Lines:    convertLines(tf),
	}
	return hunk
}

func convertLines(tf *gitdiff.TextFragment) []DiffLine {
	lines := make([]DiffLine, 0, len(tf.Lines))
	oldLineNo := int(tf.OldPosition)
	newLineNo := int(tf.NewPosition)

	for _, line := range tf.Lines {
		// Strip trailing newline from content (gitdiff includes it, our old parser didn't)
		content := strings.TrimSuffix(line.Line, "\n")

		dl := DiffLine{
			Content: content,
		}

		switch line.Op {
		case gitdiff.OpAdd:
			dl.Type = LineAdded
			dl.NewLineNo = newLineNo
			newLineNo++
		case gitdiff.OpDelete:
			dl.Type = LineRemoved
			dl.OldLineNo = oldLineNo
			oldLineNo++
		case gitdiff.OpContext:
			dl.Type = LineContext
			dl.OldLineNo = oldLineNo
			dl.NewLineNo = newLineNo
			oldLineNo++
			newLineNo++
		}

		lines = append(lines, dl)
	}

	return lines
}
