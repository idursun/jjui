package diff

import (
	"testing"

	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
)

// renderFileList is a helper that renders the file list to populate lastOutput
func renderFileList(fl *FileList) {
	dl := render.NewDisplayContext()
	box := layout.Box{R: cellbuf.Rect(0, 0, 80, 20)}
	fl.ViewRect(dl, box)
}

func TestFileList_Navigation(t *testing.T) {
	files := []*DiffFile{
		{OldPath: "file1.go", NewPath: "file1.go", Status: FileModified},
		{OldPath: "file2.go", NewPath: "file2.go", Status: FileAdded},
		{OldPath: "file3.go", NewPath: "file3.go", Status: FileDeleted},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	assert.Equal(t, 0, fl.SelectedIndex())
	assert.Equal(t, "file1.go", fl.SelectedFile().Path())

	fl.MoveDown()
	renderFileList(fl)
	assert.Equal(t, 1, fl.SelectedIndex())
	assert.Equal(t, "file2.go", fl.SelectedFile().Path())

	fl.MoveDown()
	renderFileList(fl)
	assert.Equal(t, 2, fl.SelectedIndex())

	// Can't move past the end
	fl.MoveDown()
	renderFileList(fl)
	assert.Equal(t, 2, fl.SelectedIndex())

	fl.MoveUp()
	renderFileList(fl)
	assert.Equal(t, 1, fl.SelectedIndex())

	fl.MoveUp()
	renderFileList(fl)
	assert.Equal(t, 0, fl.SelectedIndex())

	// Can't move before the start
	fl.MoveUp()
	renderFileList(fl)
	assert.Equal(t, 0, fl.SelectedIndex())
}

func TestFileList_SetSelectedIndex(t *testing.T) {
	files := []*DiffFile{
		{OldPath: "a.go", NewPath: "a.go"},
		{OldPath: "b.go", NewPath: "b.go"},
		{OldPath: "c.go", NewPath: "c.go"},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	fl.SetSelectedIndex(2)
	renderFileList(fl)
	assert.Equal(t, 2, fl.SelectedIndex())

	// Clamp to valid range
	fl.SetSelectedIndex(10)
	renderFileList(fl)
	assert.Equal(t, 2, fl.SelectedIndex())

	fl.SetSelectedIndex(-5)
	renderFileList(fl)
	assert.Equal(t, 0, fl.SelectedIndex())
}

func TestFileList_FileCount(t *testing.T) {
	fl := NewFileList([]*DiffFile{
		{OldPath: "a.go", NewPath: "a.go"},
		{OldPath: "b.go", NewPath: "b.go"},
	})

	assert.Equal(t, 2, fl.FileCount())
}

func TestFileList_EmptyList(t *testing.T) {
	fl := NewFileList(nil)
	renderFileList(fl)

	assert.Equal(t, 0, fl.FileCount())
	assert.Nil(t, fl.SelectedFile())
	assert.Equal(t, -1, fl.SelectedIndex())

	// Operations on empty list should not panic
	fl.MoveUp()
	fl.MoveDown()
	fl.SetSelectedIndex(0)
}

func TestFileStatus_String(t *testing.T) {
	assert.Equal(t, "M", FileModified.String())
	assert.Equal(t, "A", FileAdded.String())
	assert.Equal(t, "D", FileDeleted.String())
	assert.Equal(t, "R", FileRenamed.String())
	assert.Equal(t, "C", FileCopied.String())
}

func TestDiffFile_Path(t *testing.T) {
	// Prefer NewPath
	file := &DiffFile{OldPath: "old.go", NewPath: "new.go"}
	assert.Equal(t, "new.go", file.Path())

	// Fall back to OldPath
	file2 := &DiffFile{OldPath: "only.go", NewPath: ""}
	assert.Equal(t, "only.go", file2.Path())
}

func TestFileList_TreeStructure(t *testing.T) {
	files := []*DiffFile{
		{NewPath: "src/ui/diff/file_list.go", Status: FileModified},
		{NewPath: "src/ui/diff/tree.go", Status: FileAdded},
		{NewPath: "src/ui/old/viewer.go", Status: FileDeleted},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	// Should have tree nodes: src/ui/ dir, diff/ dir, old/ dir, and 3 files
	assert.Equal(t, 3, fl.FileCount())

	// First selected should be a file (navigation skips dirs)
	assert.NotNil(t, fl.SelectedFile())
	assert.GreaterOrEqual(t, fl.SelectedIndex(), 0)
}

func TestFileList_CollapseSingleChildDirs(t *testing.T) {
	files := []*DiffFile{
		{NewPath: "src/ui/diff/file.go", Status: FileModified},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	// "src" -> "ui" -> "diff" should collapse to "src/ui/diff"
	// visible should be: dir("src/ui/diff"), file("file.go")
	visible := fl.lastOutput.VisibleItems
	assert.Equal(t, 2, len(visible))
	firstNode := visible[0].Node.(*diffTreeNode)
	secondNode := visible[1].Node.(*diffTreeNode)
	assert.True(t, firstNode.isDir)
	assert.Equal(t, "src/ui/diff", firstNode.name)
	assert.False(t, secondNode.isDir)
	assert.Equal(t, "file.go", secondNode.name)
}

func TestFileList_TreeNavigation_SkipsDirs(t *testing.T) {
	files := []*DiffFile{
		{NewPath: "src/a.go", Status: FileModified},
		{NewPath: "src/b.go", Status: FileAdded},
		{NewPath: "lib/c.go", Status: FileDeleted},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	// visible order (dirs first, alphabetical):
	// 0: lib/ (dir)
	// 1:   c.go (file, fileIndex=2)
	// 2: src/ (dir)
	// 3:   a.go (file, fileIndex=0)
	// 4:   b.go (file, fileIndex=1)

	// First selected should be first file (lib/c.go, fileIndex=2)
	assert.Equal(t, 2, fl.SelectedIndex())
	assert.Equal(t, "lib/c.go", fl.SelectedFile().Path())

	// MoveDown -> next file (src/a.go, fileIndex=0)
	fl.MoveDown()
	renderFileList(fl)
	assert.Equal(t, 0, fl.SelectedIndex())

	// MoveDown -> next file (src/b.go, fileIndex=1)
	fl.MoveDown()
	renderFileList(fl)
	assert.Equal(t, 1, fl.SelectedIndex())

	// MoveDown at end -> stays
	fl.MoveDown()
	renderFileList(fl)
	assert.Equal(t, 1, fl.SelectedIndex())

	// MoveUp -> back to src/a.go
	fl.MoveUp()
	renderFileList(fl)
	assert.Equal(t, 0, fl.SelectedIndex())

	// MoveUp -> back to lib/c.go
	fl.MoveUp()
	renderFileList(fl)
	assert.Equal(t, 2, fl.SelectedIndex())
}

func TestFileList_ToggleExpand(t *testing.T) {
	files := []*DiffFile{
		{NewPath: "src/a.go", Status: FileModified},
		{NewPath: "src/b.go", Status: FileAdded},
		{NewPath: "lib/c.go", Status: FileDeleted},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	// visible order:
	// 0: lib/ (dir)
	// 1:   c.go
	// 2: src/ (dir)
	// 3:   a.go
	// 4:   b.go
	visible := fl.lastOutput.VisibleItems
	assert.Equal(t, 5, len(visible))

	// Collapse src/ dir (visible index 2)
	fl.ToggleExpand(2)
	renderFileList(fl)

	// visible should now be:
	// 0: lib/ (dir)
	// 1:   c.go
	// 2: src/ (dir, collapsed)
	visible = fl.lastOutput.VisibleItems
	assert.Equal(t, 3, len(visible))
	thirdNode := visible[2].Node.(*diffTreeNode)
	assert.True(t, thirdNode.isDir)
	assert.False(t, render.IsNodeExpanded(visible[2].Node, fl.expanded, true))

	// Expand src/ dir again
	fl.ToggleExpand(2)
	renderFileList(fl)
	visible = fl.lastOutput.VisibleItems
	assert.Equal(t, 5, len(visible))
	assert.True(t, render.IsNodeExpanded(visible[2].Node, fl.expanded, true))
}

func TestFileList_ToggleExpand_RestoresSelection(t *testing.T) {
	files := []*DiffFile{
		{NewPath: "src/a.go", Status: FileModified},
		{NewPath: "lib/c.go", Status: FileDeleted},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	// Select lib/c.go (first file)
	// visible: lib/(dir), c.go, src/(dir), a.go
	assert.Equal(t, 1, fl.SelectedIndex()) // fileIndex of c.go

	// Collapse src/ (index 2) - selection should stay on c.go
	fl.ToggleExpand(2)
	renderFileList(fl)
	assert.Equal(t, 1, fl.SelectedIndex())
}

func TestFileList_ToggleExpand_HidesSelectedFile(t *testing.T) {
	files := []*DiffFile{
		{NewPath: "src/a.go", Status: FileModified},
		{NewPath: "lib/c.go", Status: FileDeleted},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	// Select src/a.go
	fl.SetSelectedIndex(0) // fileIndex 0 = src/a.go
	renderFileList(fl)

	// Collapse src/ dir
	fl.ToggleExpand(2) // src/ is at visible index 2
	renderFileList(fl)

	// Selection should move to nearest file (lib/c.go)
	assert.Equal(t, 1, fl.SelectedIndex())
}

func TestFileList_SetSelectedIndex_ExpandsCollapsedParent(t *testing.T) {
	files := []*DiffFile{
		{NewPath: "src/a.go", Status: FileModified},
		{NewPath: "lib/c.go", Status: FileDeleted},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	// Collapse src/ dir (visible index 2)
	fl.ToggleExpand(2)
	renderFileList(fl)
	assert.Equal(t, 3, len(fl.lastOutput.VisibleItems)) // lib/, c.go, src/

	// SetSelectedIndex to file in collapsed dir should expand it
	fl.SetSelectedIndex(0) // fileIndex 0 = src/a.go
	renderFileList(fl)
	assert.Equal(t, 0, fl.SelectedIndex())
	assert.Equal(t, 4, len(fl.lastOutput.VisibleItems)) // lib/, c.go, src/, a.go
}

func TestFileList_TreeDirsFirstAlphabetical(t *testing.T) {
	files := []*DiffFile{
		{NewPath: "z.go", Status: FileModified},
		{NewPath: "a.go", Status: FileAdded},
		{NewPath: "dir_b/x.go", Status: FileModified},
		{NewPath: "dir_a/y.go", Status: FileDeleted},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	// Expected order: dir_a/(dir), y.go, dir_b/(dir), x.go, a.go(file), z.go(file)
	visible := fl.lastOutput.VisibleItems
	assert.Equal(t, 6, len(visible))
	assert.True(t, visible[0].Node.(*diffTreeNode).isDir)
	assert.Equal(t, "dir_a", visible[0].Node.(*diffTreeNode).name)
	assert.Equal(t, "y.go", visible[1].Node.(*diffTreeNode).name)
	assert.True(t, visible[2].Node.(*diffTreeNode).isDir)
	assert.Equal(t, "dir_b", visible[2].Node.(*diffTreeNode).name)
	assert.Equal(t, "x.go", visible[3].Node.(*diffTreeNode).name)
	assert.Equal(t, "a.go", visible[4].Node.(*diffTreeNode).name)
	assert.Equal(t, "z.go", visible[5].Node.(*diffTreeNode).name)
}

func TestFileList_FlatFilesNoTree(t *testing.T) {
	// Files with no directories should produce a flat list (no dir nodes)
	files := []*DiffFile{
		{NewPath: "b.go", Status: FileModified},
		{NewPath: "a.go", Status: FileAdded},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	// Files are sorted alphabetically
	visible := fl.lastOutput.VisibleItems
	assert.Equal(t, 2, len(visible))
	assert.False(t, visible[0].Node.(*diffTreeNode).isDir)
	assert.Equal(t, "a.go", visible[0].Node.(*diffTreeNode).name)
	assert.False(t, visible[1].Node.(*diffTreeNode).isDir)
	assert.Equal(t, "b.go", visible[1].Node.(*diffTreeNode).name)
}

func TestFileList_ToggleExpand_NonDir(t *testing.T) {
	files := []*DiffFile{
		{NewPath: "a.go", Status: FileModified},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	// Toggling a file node should be a no-op
	initialLen := len(fl.lastOutput.VisibleItems)
	fl.ToggleExpand(0)
	renderFileList(fl)
	assert.Equal(t, initialLen, len(fl.lastOutput.VisibleItems))
}

func TestFileList_ToggleExpand_OutOfRange(t *testing.T) {
	files := []*DiffFile{
		{NewPath: "a.go", Status: FileModified},
	}

	fl := NewFileList(files)
	renderFileList(fl)

	// Should not panic
	fl.ToggleExpand(-1)
	fl.ToggleExpand(100)
}
