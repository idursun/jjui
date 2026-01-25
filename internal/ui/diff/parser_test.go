package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse_SimpleModification(t *testing.T) {
	diffText := `diff --git a/file.go b/file.go
index abc123..def456 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main

+import "fmt"
 func main() {
`

	result := Parse(diffText)

	assert.Len(t, result.Files, 1)
	file := result.Files[0]
	assert.Equal(t, "file.go", file.OldPath)
	assert.Equal(t, "file.go", file.NewPath)
	assert.Equal(t, FileModified, file.Status)
	assert.False(t, file.IsBinary)

	assert.Len(t, file.Hunks, 1)
	hunk := file.Hunks[0]
	assert.Equal(t, 1, hunk.OldStart)
	assert.Equal(t, 3, hunk.OldCount)
	assert.Equal(t, 1, hunk.NewStart)
	assert.Equal(t, 4, hunk.NewCount)

	assert.Len(t, hunk.Lines, 4)
	assert.Equal(t, LineContext, hunk.Lines[0].Type)
	assert.Equal(t, "package main", hunk.Lines[0].Content)
	assert.Equal(t, LineContext, hunk.Lines[1].Type)
	assert.Equal(t, LineAdded, hunk.Lines[2].Type)
	assert.Equal(t, `import "fmt"`, hunk.Lines[2].Content)
	assert.Equal(t, LineContext, hunk.Lines[3].Type)
}

func TestParse_NewFile(t *testing.T) {
	diffText := `diff --git a/newfile.go b/newfile.go
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/newfile.go
@@ -0,0 +1,3 @@
+package main
+
+func foo() {}
`

	result := Parse(diffText)

	assert.Len(t, result.Files, 1)
	file := result.Files[0]
	assert.Equal(t, FileAdded, file.Status)
	assert.Equal(t, "newfile.go", file.Path())

	assert.Len(t, file.Hunks, 1)
	assert.Len(t, file.Hunks[0].Lines, 3)
	for _, line := range file.Hunks[0].Lines {
		assert.Equal(t, LineAdded, line.Type)
	}
}

func TestParse_DeletedFile(t *testing.T) {
	diffText := `diff --git a/oldfile.go b/oldfile.go
deleted file mode 100644
index abc1234..0000000
--- a/oldfile.go
+++ /dev/null
@@ -1,2 +0,0 @@
-package main
-func old() {}
`

	result := Parse(diffText)

	assert.Len(t, result.Files, 1)
	file := result.Files[0]
	assert.Equal(t, FileDeleted, file.Status)

	assert.Len(t, file.Hunks, 1)
	for _, line := range file.Hunks[0].Lines {
		assert.Equal(t, LineRemoved, line.Type)
	}
}

func TestParse_RenamedFile(t *testing.T) {
	diffText := `diff --git a/old.go b/new.go
similarity index 95%
rename from old.go
rename to new.go
index abc123..def456 100644
--- a/old.go
+++ b/new.go
@@ -1,3 +1,3 @@
 package main

-func old() {}
+func new() {}
`

	result := Parse(diffText)

	assert.Len(t, result.Files, 1)
	file := result.Files[0]
	assert.Equal(t, FileRenamed, file.Status)
	assert.Equal(t, "old.go", file.OldPath)
	assert.Equal(t, "new.go", file.NewPath)
}

func TestParse_MultipleFiles(t *testing.T) {
	diffText := `diff --git a/file1.go b/file1.go
--- a/file1.go
+++ b/file1.go
@@ -1 +1 @@
-old
+new
diff --git a/file2.go b/file2.go
--- a/file2.go
+++ b/file2.go
@@ -1 +1 @@
-foo
+bar
`

	result := Parse(diffText)

	assert.Len(t, result.Files, 2)
	assert.Equal(t, "file1.go", result.Files[0].Path())
	assert.Equal(t, "file2.go", result.Files[1].Path())
}

func TestParse_LineNumbers(t *testing.T) {
	diffText := `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -10,4 +10,5 @@
 line 10
 line 11
+new line
 line 12
 line 13
`

	result := Parse(diffText)

	assert.Len(t, result.Files, 1)
	hunk := result.Files[0].Hunks[0]

	// Context line at old:10, new:10
	assert.Equal(t, 10, hunk.Lines[0].OldLineNo)
	assert.Equal(t, 10, hunk.Lines[0].NewLineNo)

	// Context line at old:11, new:11
	assert.Equal(t, 11, hunk.Lines[1].OldLineNo)
	assert.Equal(t, 11, hunk.Lines[1].NewLineNo)

	// Added line - no old line number, new:12
	assert.Equal(t, 0, hunk.Lines[2].OldLineNo)
	assert.Equal(t, 12, hunk.Lines[2].NewLineNo)

	// Context line at old:12, new:13
	assert.Equal(t, 12, hunk.Lines[3].OldLineNo)
	assert.Equal(t, 13, hunk.Lines[3].NewLineNo)
}

func TestParse_EmptyDiff(t *testing.T) {
	result := Parse("")
	assert.Len(t, result.Files, 0)
}

func TestParse_HunkWithContext(t *testing.T) {
	diffText := `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -1,3 +1,3 @@ func main()
 line1
-old
+new
 line3
`

	result := Parse(diffText)

	assert.Len(t, result.Files, 1)
	hunk := result.Files[0].Hunks[0]
	assert.Equal(t, "func main()", hunk.Header)
}
