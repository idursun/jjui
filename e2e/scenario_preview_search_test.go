//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Preview_UpdatesWhenCyclingSearchResults(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t).
		Append("README.md", "alpha-one-line\n").
		Commit("alpha-one").
		Append("README.md", "alpha-two-line\n").
		Commit("alpha-two")

	jjuiBin := jjuiBinary(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBin, repo, "alpha-two")
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyP, "p", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "alpha-two-line")
	}); err != nil {
		t.Fatalf("preview did not render the first matching revision: %v", err)
	}

	if err := session.SendKey(ghostty.KeySlash, "/", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendText("alpha"); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForStableScreen(ctx, 3, func(screen []string) bool {
		return screenContains(screen, "quick search") && screenContains(screen, "alpha-two-line")
	}); err != nil {
		t.Fatalf("quick search did not become active: %v", err)
	}
	if err := session.SendKey(ghostty.KeyQuote, "'", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "alpha-one-line") && !screenContains(screen, "alpha-two-line")
	}); err != nil {
		t.Fatalf("preview did not update when cycling to the next search result: %v", err)
	}

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
