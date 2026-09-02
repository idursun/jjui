//go:build e2e

package main

import (
	"strings"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Revisions_AbandonsConflictedRevision(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t).
		Append("README.md", "left\n").
		Commit("left").
		Bookmark("left", "@-").
		New("@--").
		Append("README.md", "right\n").
		Commit("right").
		Bookmark("right", "@-").
		New("left", "right")
	conflictChangeID := strings.TrimSpace(runCommand(t, repo.Path(), repo.Env(), "jj", "log", "-r", "@", "-T", "change_id", "--no-graph", "--color", "never"))

	jjuiBin := jjuiBinary(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBin, repo, "left")
	if err := session.SendKey(ghostty.KeyA, "a", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "abandon") && screenContains(screen, "enter")
	}); err != nil {
		t.Fatalf("abandon confirmation did not open for the conflicted revision: %v", err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}

	waitFor(t, ctx, func() (bool, error) {
		log := runCommand(t, repo.Path(), repo.Env(), "jj", "log", "-r", "all()", "-T", "change_id", "--no-graph", "--color", "never")
		return !strings.Contains(log, conflictChangeID), nil
	})

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
