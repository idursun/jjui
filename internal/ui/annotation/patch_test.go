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

func TestParseGitPatchPreservesPathsFromGitMetadata(t *testing.T) {
	tests := []struct {
		name    string
		patch   string
		oldPath string
		newPath string
		summary string
	}{
		{
			name: "unquoted spaces with content markers",
			patch: "diff --git a/hello world.txt b/hello world.txt\n" +
				"--- a/hello world.txt\t\n" +
				"+++ b/hello world.txt\t\n" +
				"@@ -1 +1 @@\n-old\n+new",
			oldPath: "hello world.txt",
			newPath: "hello world.txt",
		},
		{
			name: "header-only file with spaces",
			patch: "diff --git a/hello world.txt b/hello world.txt\n" +
				"new file mode 100644\n" +
				"index 0000000000..e69de29bb2",
			newPath: "hello world.txt",
		},
		{
			name: "header-only file containing apparent separator",
			patch: "diff --git a/foo b/bar.txt b/foo b/bar.txt\n" +
				"new file mode 100644\n" +
				"index 0000000000..e69de29bb2",
			newPath: "foo b/bar.txt",
		},
		{
			name: "header-only deleted file with spaces",
			patch: "diff --git a/hello world.txt b/hello world.txt\n" +
				"deleted file mode 100644\n" +
				"index e69de29bb2..0000000000",
			oldPath: "hello world.txt",
		},
		{
			name: "mode-only file containing apparent separator",
			patch: "diff --git a/foo b/bar.txt b/foo b/bar.txt\n" +
				"old mode 100644\n" +
				"new mode 100755",
			oldPath: "foo b/bar.txt",
			newPath: "foo b/bar.txt",
		},
		{
			name: "binary file containing apparent separator",
			patch: "diff --git a/foo b/image.png b/foo b/image.png\n" +
				"index 1111111111..2222222222 100644\n" +
				"Binary files a/foo b/image.png and b/foo b/image.png differ",
			oldPath: "foo b/image.png",
			newPath: "foo b/image.png",
		},
		{
			name: "rename with spaces",
			patch: "diff --git a/old name.txt b/new name.txt\n" +
				"similarity index 100%\n" +
				"rename from old name.txt\n" +
				"rename to new name.txt",
			oldPath: "old name.txt",
			newPath: "new name.txt",
			summary: "old name.txt -> new name.txt",
		},
		{
			name: "rename containing apparent separator",
			patch: "diff --git a/foo b/old name.txt b/foo b/new name.txt\n" +
				"similarity index 100%\n" +
				"rename from foo b/old name.txt\n" +
				"rename to foo b/new name.txt",
			oldPath: "foo b/old name.txt",
			newPath: "foo b/new name.txt",
			summary: "foo b/old name.txt -> foo b/new name.txt",
		},
		{
			name: "quoted path",
			patch: "diff --git \"a/tab\\tname.txt\" \"b/tab\\tname.txt\"\n" +
				"--- \"a/tab\\tname.txt\"\n" +
				"+++ \"b/tab\\tname.txt\"\n" +
				"@@ -1 +1 @@\n-old\n+new",
			oldPath: "tab\tname.txt",
			newPath: "tab\tname.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := parseGitPatch(tt.patch)

			require.Len(t, files, 1)
			assert.Equal(t, tt.oldPath, files[0].OldPath)
			assert.Equal(t, tt.newPath, files[0].NewPath)
			if tt.summary != "" {
				require.NotEmpty(t, files[0].Lines)
				assert.Equal(t, tt.summary, files[0].Lines[0].Content)
			}
		})
	}
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
