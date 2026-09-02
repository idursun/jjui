//go:build e2e

package main

import (
	"path/filepath"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Git_PushesBookmarkToFileRemote(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	root := filepath.Dir(repo.Path())
	remote := filepath.Join(root, "remote.git")

	// A bare repository gives the scenario a real Git remote without requiring
	// a network service. The same setup works inside the Docker test image.
	runCommand(t, root, repo.Env(), "git", "init", "--bare", remote)
	repo.
		JJ("git", "remote", "add", "origin", remote).
		JJ("describe", "-m", "remote main").
		Bookmark("main", "@")

	jjuiBin := jjuiBinary(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBin, repo, "initial")

	// Open Git Operations, select the push actions, and push the bookmark.
	if err := session.SendKey(ghostty.KeyG, "g", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "origin")
	}); err != nil {
		t.Fatalf("git operations menu did not show origin: %v", err)
	}
	if err := session.SendKey(ghostty.KeyP, "p", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "git push")
	}); err != nil {
		t.Fatalf("git push actions did not render: %v", err)
	}
	if err := session.SendKey(ghostty.KeyB, "b", 0); err != nil {
		t.Fatal(err)
	}

	waitFor(t, ctx, func() (bool, error) {
		_, err := commandOutput(root, repo.Env(), "git", "--git-dir", remote, "show-ref", "--verify", "--quiet", "refs/heads/main")
		return err == nil, nil
	})

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
