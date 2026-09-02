//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Preview_UpdatesWhenSquashSelectsTarget(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t).
		Append("README.md", "source-only-line\n").
		Commit("source")

	jjuiBin := jjuiBinary(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBin, repo, "source")
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyP, "p", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "source-only-line")
	}); err != nil {
		t.Fatalf("preview did not render the source revision: %v", err)
	}

	if err := session.SendKey(ghostty.KeyS, "S", ghostty.ModShift); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "<< into >>") && screenContains(screen, "created by the PTY test") && !screenContains(screen, "source-only-line")
	}); err != nil {
		t.Fatalf("preview did not update to the squash target: %v", err)
	}

	if err := session.SendKey(ghostty.KeyEscape, "", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
