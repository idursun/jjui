//go:build e2e

package main

import (
	"strings"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Diff_PreservesTabIndentation(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t).
		Append("README.md", "\tindented line\n").
		Commit("tabs")

	jjuiBin := jjuiBinary(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBin, repo, "tabs")
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyP, "p", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "indented line")
	}); err != nil {
		t.Fatalf("preview did not render the tab-indented line: %v", err)
	}

	if err := session.SendKey(ghostty.KeyD, "d", 0); err != nil {
		t.Fatal(err)
	}
	screen, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "diff      ↑/k") && screenContains(screen, "indented line")
	})
	if err != nil {
		t.Fatalf("d did not open the diff view: %v", err)
	}
	row := screen[screenRowIndexContaining(screen, "indented line")]
	indent := strings.Index(row, "indented line")
	if indent < 8 {
		t.Fatalf("tab indentation was lost in diff view: line starts at column %d\nrow: %q\nscreen:\n%s", indent, row, formatScreen(screen))
	}

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
