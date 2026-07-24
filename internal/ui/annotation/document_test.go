package annotation

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeletedFileFullPresentationUsesOldCoordinates(t *testing.T) {
	file := fileItem{
		Path:    "deleted.go",
		Patch:   &patchFile{OldPath: "deleted.go"},
		Content: []string{"first", "removed"},
	}
	source := reviewSource{file: &file, presentation: filePresentation}

	oldLines, newLines, snippet := source.AnnotationLocation(1, 1)
	assert.Equal(t, lineRange{Start: 2, End: 2}, oldLines)
	assert.Empty(t, newLines)
	assert.Equal(t, " removed", snippet)

	annotation := Annotation{OldLines: oldLines}
	assert.True(t, source.Contains(1, annotation))
	assert.False(t, source.Contains(1, Annotation{NewLines: oldLines}))
	assert.Equal(t, 1, mustAnchor(t, source, annotation))

	diffFile := fileItem{
		Path: "deleted.go",
		Patch: &patchFile{
			OldPath: "deleted.go",
			Lines: []patchLine{
				{Kind: lineHunk},
				{Kind: lineRemoved, OldLine: 2},
			},
		},
	}
	diffSource := reviewSource{file: &diffFile, presentation: diffPresentation}
	assert.True(t, diffSource.Contains(1, annotation))
	assert.Equal(t, 1, mustAnchor(t, diffSource, annotation))
}

func TestFullPresentationKeepsNewCoordinatesForNonDeletedFiles(t *testing.T) {
	file := fileItem{
		Path:    "changed.go",
		Patch:   &patchFile{NewPath: "changed.go"},
		Content: []string{"first", "changed"},
	}
	source := reviewSource{file: &file, presentation: filePresentation}

	oldLines, newLines, _ := source.AnnotationLocation(1, 1)
	assert.Empty(t, oldLines)
	assert.Equal(t, lineRange{Start: 2, End: 2}, newLines)
	assert.True(t, source.Contains(1, Annotation{NewLines: newLines}))
	assert.Equal(t, 1, mustAnchor(t, source, Annotation{NewLines: newLines}))
}

func TestDeletedFileAnnotationRendersInFullPresentation(t *testing.T) {
	file := fileItem{
		Path:    "deleted.go",
		Patch:   &patchFile{OldPath: "deleted.go"},
		Content: []string{"first", "removed"},
	}
	document := reviewDocument{
		revision:     "change",
		files:        []fileItem{file},
		presentation: filePresentation,
	}
	annotations := annotationStore{}
	annotations.Add(Annotation{
		ChangeID: "change",
		File:     "deleted.go",
		OldLines: lineRange{Start: 2, End: 2},
		Comment:  "review removed line",
	})

	renderer := newAnnotationRenderer(false)
	lines, _ := renderer.buildFileLines(annotationViewState{
		document:    &document,
		annotations: &annotations,
	}, file, 80)

	var rendered string
	for _, line := range lines {
		rendered += ansi.Strip(line.Content) + "\n"
	}
	assert.Contains(t, rendered, "comment: review removed line")
}

func mustAnchor(t *testing.T, source reviewSource, annotation Annotation) int {
	t.Helper()
	index, ok := source.Anchor(annotation)
	require.True(t, ok)
	return index
}
