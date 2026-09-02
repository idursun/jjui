//go:build e2e

package main

import (
	"strings"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Revisions_RebasesSourceWithInsertBetween(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t).
		Append("README.md", "base\n").
		Commit("base").
		Append("README.md", "source\n").
		Commit("source").
		New("@--").
		Append("README.md", "target\n").
		Commit("target")

	jjuiBin := jjuiBinary(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBin, repo, "target")

	// The visible order is the working-copy child, target, then the source
	// branch. Select source and move back to target for the insert target.
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForStableScreen(ctx, 3, func(screen []string) bool {
		return screenContains(screen, "target")
	}); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForStableScreen(ctx, 3, func(screen []string) bool {
		return screenContains(screen, "source")
	}); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeySpace, " ", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyK, "k", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyR, "r", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "rebase")
	}); err != nil {
		t.Fatalf("rebase operation did not open: %v", err)
	}

	if err := session.SendKey(ghostty.KeyS, "s", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyI, "i", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "insert") && screenContains(screen, "between")
	}); err != nil {
		t.Fatalf("rebase insert-between mode did not render: %v", err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}

	waitFor(t, ctx, func() (bool, error) {
		log := runCommand(t, repo.Path(), repo.Env(), "jj", "log", "-r", "all()", "-T", "description ++ \"\\n\"", "--no-graph", "--color", "never")
		base := strings.Index(log, "base\n")
		source := strings.Index(log, "source\n")
		target := strings.Index(log, "target\n")
		return target >= 0 && source > target && base > source, nil
	})

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
