//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Git_GOpensTheRemotesAndPushes(t *testing.T) {
	t.Parallel()
	repo, remote := newRepoWithFileRemote(t)
	repo.JJ("bookmark", "create", "gone", "-r", "@-")
	repo.JJ("bookmark", "track", "gone", "--remote", "origin")
	repo.JJ("git", "push", "--bookmark", "gone")
	repo.JJ("bookmark", "delete", "gone")

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "initial")
	send := func(key ghostty.Key, text string) {
		t.Helper()
		if err := session.SendKey(key, text, 0); err != nil {
			t.Fatal(err)
		}
	}
	send(ghostty.KeyG, "g")
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool { return screenContains(screen, "back to log") }); err != nil {
		t.Fatalf("g did not focus the remotes: %v", err)
	}
	send(ghostty.KeyL, "l")
	send(ghostty.KeyL, "l")
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool { return screenContains(screen, `remote=exact:"origin"`) }); err != nil {
		t.Fatalf("l did not move to origin: %v", err)
	}
	send(ghostty.KeyD, "d")
	waitFor(t, ctx, func() (bool, error) {
		return !remoteHasBranch(t, repo, remote, "gone"), nil
	})
}
