//go:build e2e

package main

import (
	"strings"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Bookmarks_DeletesSpecialName(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	const bookmark = "1.3.63-+-json-length-fix"
	commitID := strings.TrimSpace(runCommand(t, repo.Path(), repo.Env(), "jj", "log", "-r", "@-", "-T", "commit_id", "--no-graph", "--color", "never"))
	runCommand(t, repo.Path(), repo.Env(), "git", "branch", bookmark, commitID)
	repo.JJ("git", "import")

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "initial")
	if err := session.SendKey(ghostty.KeyB, "b", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, bookmark)
	}); err != nil {
		t.Fatalf("bookmark menu did not render the special bookmark name: %v\nraw output:\n%s", err, session.RawOutput())
	}

	if err := session.SendKey(ghostty.KeyD, "d", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "delete '"+bookmark+"'")
	}); err != nil {
		t.Fatalf("bookmark delete filter did not render the special bookmark: %v\nraw output:\n%s", err, session.RawOutput())
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}

	waitFor(t, ctx, func() (bool, error) {
		return !bookmarkExists(t, repo.Path(), repo.Env(), bookmark), nil
	})

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v\nraw output:\n%s", err, session.RawOutput())
	}
}
