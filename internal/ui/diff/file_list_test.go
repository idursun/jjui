package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileList_Navigation(t *testing.T) {
	files := []*DiffFile{
		{OldPath: "file1.go", NewPath: "file1.go", Status: FileModified},
		{OldPath: "file2.go", NewPath: "file2.go", Status: FileAdded},
		{OldPath: "file3.go", NewPath: "file3.go", Status: FileDeleted},
	}

	fl := NewFileList(files)

	assert.Equal(t, 0, fl.SelectedIndex())
	assert.Equal(t, "file1.go", fl.SelectedFile().Path())

	fl.MoveDown()
	assert.Equal(t, 1, fl.SelectedIndex())
	assert.Equal(t, "file2.go", fl.SelectedFile().Path())

	fl.MoveDown()
	assert.Equal(t, 2, fl.SelectedIndex())

	// Can't move past the end
	fl.MoveDown()
	assert.Equal(t, 2, fl.SelectedIndex())

	fl.MoveUp()
	assert.Equal(t, 1, fl.SelectedIndex())

	fl.MoveUp()
	assert.Equal(t, 0, fl.SelectedIndex())

	// Can't move before the start
	fl.MoveUp()
	assert.Equal(t, 0, fl.SelectedIndex())
}

func TestFileList_SetSelectedIndex(t *testing.T) {
	files := []*DiffFile{
		{OldPath: "a.go", NewPath: "a.go"},
		{OldPath: "b.go", NewPath: "b.go"},
		{OldPath: "c.go", NewPath: "c.go"},
	}

	fl := NewFileList(files)

	fl.SetSelectedIndex(2)
	assert.Equal(t, 2, fl.SelectedIndex())

	// Clamp to valid range
	fl.SetSelectedIndex(10)
	assert.Equal(t, 2, fl.SelectedIndex())

	fl.SetSelectedIndex(-5)
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

	assert.Equal(t, 0, fl.FileCount())
	assert.Nil(t, fl.SelectedFile())
	assert.Equal(t, 0, fl.SelectedIndex())

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
