package jj

import (
	"fmt"
	"path/filepath"
	"strings"
)

// FileName is a canonical repository-relative path returned to and accepted by jj.
type FileName struct {
	path string
}

func NewFileName(path string) FileName {
	return FileName{path: path}
}

func (f FileName) Path() string {
	return f.path
}

func (f FileName) IsEmpty() bool {
	return f.path == ""
}

// Display returns the path relative to the directory where jjui was started.
func (f FileName) Display(repoRoot, workingDirectory string) string {
	if f.path == "" || repoRoot == "" || workingDirectory == "" {
		return f.path
	}
	directory := strings.HasSuffix(f.path, "/")
	absPath := filepath.Join(repoRoot, filepath.FromSlash(f.path))
	rel, err := filepath.Rel(workingDirectory, absPath)
	if err != nil {
		return f.path
	}
	rel = filepath.ToSlash(rel)
	if directory {
		if rel == "." {
			return "./"
		}
		if !strings.HasSuffix(rel, "/") {
			rel += "/"
		}
	}
	return rel
}

// Escaped returns a jj fileset expression for this path.
func (f FileName) Escaped() string {
	path := strings.ReplaceAll(f.path, `\`, `\\`)
	path = strings.ReplaceAll(path, `"`, `\"`)
	return fmt.Sprintf(`file:"%s"`, path)
}

// ShellEscaped returns this path as a single shell argument.
func (f FileName) ShellEscaped() string {
	return "'" + strings.ReplaceAll(f.path, "'", "'\\''") + "'"
}
