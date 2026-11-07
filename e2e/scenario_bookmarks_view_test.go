//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_BookmarksView_FiltersAndLeaves(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	repo.JJ("describe", "-m", "alpha work")
	repo.JJ("bookmark", "create", "alpha-feature", "-r", "@")
	repo.JJ("new", "-m", "beta work")
	repo.JJ("bookmark", "create", "beta-feature", "-r", "@")

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "beta work")
	if err := session.SendKey(ghostty.KeyB, "b", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "alpha-feature") && screenContains(screen, "beta-feature")
	}); err != nil {
		t.Fatalf("bookmarks view did not list both bookmarks: %v", err)
	}

	if err := session.SendKey(ghostty.KeySlash, "/", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendText("alpha"); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "\r", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "alpha-feature") && !screenContains(screen, "beta-feature")
	}); err != nil {
		t.Fatalf("filter did not narrow the bookmarks: %v\nraw output:\n%s", err, session.RawOutput())
	}

	// With only the local bookmarks in play, tab cycles all -> local and keeps the filter.
	if err := session.SendKey(ghostty.KeyTab, "\t", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "alpha-feature") && !screenContains(screen, "beta-feature")
	}); err != nil {
		t.Fatalf("local remote lost the filter: %v", err)
	}

	if err := session.SendKey(ghostty.KeyEscape, "\x1b", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "beta work")
	}); err != nil {
		t.Fatalf("esc did not restore the previous revset: %v", err)
	}
}
