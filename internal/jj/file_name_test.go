package jj

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileNameDisplay(t *testing.T) {
	repo := filepath.Join(string(filepath.Separator), "work", "repo")
	tests := []struct {
		name string
		path string
		cwd  string
		want string
	}{
		{name: "repository root", path: "src/main.go", cwd: repo, want: "src/main.go"},
		{name: "nested descendant", path: "src/pkg/main.go", cwd: filepath.Join(repo, "src"), want: "pkg/main.go"},
		{name: "nested sibling", path: "docs/readme.md", cwd: filepath.Join(repo, "src"), want: "../docs/readme.md"},
		{name: "nested ancestor directory", path: "src/", cwd: filepath.Join(repo, "src", "pkg"), want: "../"},
		{name: "startup directory", path: "src/", cwd: filepath.Join(repo, "src"), want: "./"},
		{name: "outside repository", path: "../shared/file.go", cwd: repo, want: "../shared/file.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewFileName(tt.path).Display(repo, tt.cwd))
		})
	}
}

func TestFileNameDisplayFallbacks(t *testing.T) {
	assert.Equal(t, "", NewFileName("").Display("/repo", "/repo"))
	assert.Equal(t, "src/main.go", NewFileName("src/main.go").Display("", "/repo"))
	assert.Equal(t, "src/main.go", NewFileName("src/main.go").Display("/repo", ""))
}

func TestFileNameEscaped(t *testing.T) {
	tests := map[string]string{
		"a b.go":      `file:"a b.go"`,
		"a'b.go":      `file:"a'b.go"`,
		`a"quote.go`:  `file:"a\"quote.go"`,
		`dir\file.go`: `file:"dir\\file.go"`,
		`both\".go`:   `file:"both\\\".go"`,
	}
	for path, want := range tests {
		assert.Equal(t, want, NewFileName(path).Escaped())
	}
}

func TestFileNameShellEscaped(t *testing.T) {
	tests := map[string]string{
		"a b.go":  `'a b.go'`,
		"a'b.go":  `'a'\''b.go'`,
		`a"b.go`:  `'a"b.go'`,
		`a$b.go`:  `'a$b.go'`,
		`a; b.go`: `'a; b.go'`,
		`a\b.go`:  `'a\b.go'`,
	}
	for path, want := range tests {
		assert.Equal(t, want, NewFileName(path).ShellEscaped())
	}
}

func TestFileNameAccessors(t *testing.T) {
	assert.True(t, NewFileName("").IsEmpty())
	assert.False(t, NewFileName("a.go").IsEmpty())
	assert.Equal(t, "a.go", NewFileName("a.go").Path())
}
