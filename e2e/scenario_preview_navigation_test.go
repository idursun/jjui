//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Preview_UpdatesWhenNavigatingRevisions(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t).
		Append("README.md", "first-revision-line\n").
		Commit("first-revision").
		Append("README.md", "second-revision-line\n").
		Commit("second-revision")

	jjuiBin := jjuiBinary(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBin, repo, "second-revision")
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyP, "p", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "second-revision-line")
	}); err != nil {
		t.Fatalf("preview did not render the selected revision: %v", err)
	}

	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "first-revision-line") && !screenContains(screen, "second-revision-line")
	}); err != nil {
		t.Fatalf("preview did not update after moving to the parent revision: %v", err)
	}

	if err := session.SendKey(ghostty.KeyK, "k", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		// The second revision's diff includes the first line as unchanged
		// context, so the presence of first-revision-line is expected here.
		// The preceding assertion established that the first preview did not
		// contain second-revision-line, making its return sufficient evidence
		// that the preview updated.
		return screenContains(screen, "second-revision-line")
	}); err != nil {
		t.Fatalf("preview did not update after moving back to the child revision: %v", err)
	}

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
