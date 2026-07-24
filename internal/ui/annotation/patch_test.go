package annotation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitPatchBuildsFilesHunksAndLineNumbers(t *testing.T) {
	files := parseGitPatch(`diff --git a/a.go b/a.go
index 1111111..2222222 100644
--- a/a.go
+++ b/a.go
@@ -10,3 +10,4 @@
 same
-old value
+new value
+extra
 tail`)

	require.Len(t, files, 1)
	assert.Equal(t, "a.go", files[0].path())
	require.Len(t, files[0].Lines, 6)
	assert.Equal(t, lineHunk, files[0].Lines[0].Kind)
	assert.Equal(t, 10, files[0].Lines[1].OldLine)
	assert.Equal(t, 10, files[0].Lines[1].NewLine)
	assert.Equal(t, 11, files[0].Lines[2].OldLine)
	assert.Equal(t, 0, files[0].Lines[2].NewLine)
	assert.Equal(t, 11, files[0].Lines[3].NewLine)
	assert.Equal(t, "new value", files[0].Lines[2].PairContent)
	assert.Equal(t, "old value", files[0].Lines[3].PairContent)
	assert.Equal(t, 12, files[0].Lines[4].NewLine)
	assert.Equal(t, 12, files[0].Lines[5].OldLine)
	assert.Equal(t, 13, files[0].Lines[5].NewLine)
}

func TestParseGitPatchHandlesQuotedPathsAndDeletion(t *testing.T) {
	files := parseGitPatch(`diff --git "a/a file.go" "b/a file.go"
deleted file mode 100644
--- "a/a file.go"
+++ /dev/null
@@ -1 +0,0 @@
-old`)

	require.Len(t, files, 1)
	assert.Equal(t, "a file.go", files[0].OldPath)
	assert.Empty(t, files[0].NewPath)
	assert.Equal(t, "a file.go", files[0].path())
	require.Len(t, files[0].Lines, 3)
	assert.Equal(t, lineMetadata, files[0].Lines[0].Kind)
	assert.Equal(t, lineRemoved, files[0].Lines[2].Kind)
}

func TestParseGitPatchCollapsesRenameMetadataToJjStylePath(t *testing.T) {
	files := parseGitPatch(`diff --git a/old.go b/new.go
rename from old.go
rename to new.go`)

	require.Len(t, files, 1)
	assert.Equal(t, "old.go", files[0].OldPath)
	assert.Equal(t, "new.go", files[0].NewPath)
	require.Len(t, files[0].Lines, 1)
	assert.Equal(t, lineMetadata, files[0].Lines[0].Kind)
	assert.Equal(t, "old.go -> new.go", files[0].Lines[0].Content)
}

func TestParseGitPatchDoesNotPairUnrelatedChangedLines(t *testing.T) {
	files := parseGitPatch(`diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,2 +1,3 @@
-old alpha
-old beta
+new gamma
+new delta
+new epsilon`)

	for _, line := range files[0].Lines {
		assert.Empty(t, line.PairContent)
	}
}

func TestParseGitPatchPairsReorderedLinesByContent(t *testing.T) {
	files := parseGitPatch(`diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,2 +1,2 @@
-first one
-second two
+second two changed
+first one changed`)

	assert.Equal(t, "first one changed", files[0].Lines[1].PairContent)
	assert.Equal(t, "second two changed", files[0].Lines[2].PairContent)
	assert.Equal(t, "second two", files[0].Lines[3].PairContent)
	assert.Equal(t, "first one", files[0].Lines[4].PairContent)
}

func TestPatchFilesOnlyIncludesChangedFiles(t *testing.T) {
	patch := []patchFile{
		{NewPath: "changed.go"},
		{OldPath: "deleted.go"},
	}

	files := patchFiles(patch)

	require.Len(t, files, 2)
	assert.Equal(t, "changed.go", files[0].Path)
	assert.Equal(t, "deleted.go", files[1].Path)
	assert.NotNil(t, files[0].Patch)
}
