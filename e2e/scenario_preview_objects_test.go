package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_PreviewObjects_FollowsFocusedBookmark(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	repo.JJ("describe", "-m", "bookmarked")
	repo.JJ("bookmark", "create", "focus-me", "-r", "@")

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "focus-me")
	focusTextOnRow(t, session, ctx, "focus-me")
	if err := session.SendKey(ghostty.KeyP, "p", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool { return screenContains(screen, "jj diff --color always --from") }); err != nil {
		t.Fatalf("bookmark preview command not shown: %v", err)
	}
	if err := session.SendKey(ghostty.KeyL, "l", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool { return screenContains(screen, "jj show --color always") }); err != nil {
		t.Fatalf("preview did not follow focus off the bookmark: %v", err)
	}
}

func Test_PreviewObjects_ConflictListsConflictedFiles(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	repo.Write("clash.txt", "base\n")
	repo.JJ("bookmark", "create", "base", "-r", "@")
	repo.JJ("new", "base", "-m", "left")
	repo.Write("clash.txt", "left\n")
	repo.JJ("bookmark", "create", "left", "-r", "@")
	repo.JJ("new", "base", "-m", "right")
	repo.Write("clash.txt", "right\n")
	repo.JJ("new", "left", "@", "-m", "merge")

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "(conflict)")
	focusTextOnRow(t, session, ctx, "(conflict)")
	if err := session.SendKey(ghostty.KeyP, "p", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool { return screenContains(screen, "2-sided conflict") }); err != nil {
		t.Fatalf("conflicted files not listed: %v", err)
	}
}
