//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_RevisionDetails_RenderBeforeElidedRevisionMarker(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t).
		Append("README.md", "second commit\n").
		Commit("second").
		Bookmark("main", "@-")
	writeJJUIConfig(t, repo.Env(), "[revisions]\nrevset = \"bookmarks() | @ | immutable_heads()\"\n")

	jjuiBin := jjuiBinary(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBin, repo, "second")

	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "elided revisions")
	}); err != nil {
		t.Fatalf("configured log did not render an elided revision marker: %v", err)
	}
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyL, "l", 0); err != nil {
		t.Fatal(err)
	}

	screen, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "README.md") && screenContains(screen, "elided revisions")
	})
	if err != nil {
		t.Fatalf("l did not open details above the elided marker: %v", err)
	}
	fileRow := screenRowIndexContaining(screen, "README.md")
	markerRow := screenRowIndexContaining(screen, "elided revisions")
	if fileRow >= markerRow {
		t.Fatalf("details row = %d, elided marker row = %d; want details before marker\nscreen:\n%s", fileRow, markerRow, formatScreen(screen))
	}

	if err := session.SendKey(ghostty.KeyEscape, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForStableScreen(ctx, 3, func(screen []string) bool {
		return screenContains(screen, "second") && !screenContains(screen, "README.md")
	}); err != nil {
		t.Fatalf("escape did not close revision details: %v", err)
	}
	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
