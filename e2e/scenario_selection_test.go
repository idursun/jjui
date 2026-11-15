//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

// newer's parent is initial but the next row is older, so this tells moving down apart from jumping to the parent.
func Test_Revisions_SelectionMovesCursorDown(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.repo.
		Append("README.md", "older\n").
		Commit("older").
		New("@--").
		Write("other.txt", "newer\n").
		Commit("newer")

	h.Start("older")
	h.WaitTextInOrder("newer", "older", "initial")

	h.Key("j")
	h.Key(" ")
	waitForCheckedCount(h, 1)
	h.Key(" ")
	waitForCheckedCount(h, 2)

	h.Quit()
}

func Test_Revisions_SelectionClearsWithEscape(t *testing.T) {
	t.Parallel()
	_, session, ctx := startJJUITest(t, "initial")

	if err := session.SendKey(ghostty.KeySpace, " ", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, screenContainsCheckedCount); err != nil {
		t.Fatalf("space did not select the focused revision: %v\nraw output:\n%s", err, session.RawOutput())
	}

	if err := session.SendKey(ghostty.KeyEscape, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForStableScreen(ctx, 3, func(screen []string) bool {
		return screenContains(screen, "initial") && !screenContainsCheckedCount(screen)
	}); err != nil {
		t.Fatalf("escape did not clear the revision selection: %v\nraw output:\n%s", err, session.RawOutput())
	}

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v\nraw output:\n%s", err, session.RawOutput())
	}
}

// Checking a bookmark jumps to the next bookmark, so a second space checks that one instead of a revision.
func Test_Revisions_SelectionOnObjectJumpsToNextOfKind(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.repo.
		Append("README.md", "first\n").
		Commit("first").
		Bookmark("first-bm", "@-").
		Append("README.md", "middle\n").
		Commit("middle").
		Append("README.md", "second\n").
		Commit("second").
		Bookmark("second-bm", "@-")

	h.Start("second")
	h.Key("j")
	h.Key("i")
	h.Key("b")
	h.Key(" ")
	waitForCheckedCount(h, 1)
	h.Key(" ")
	waitForCheckedCount(h, 2)
	h.Key("x")
	waitFor(h.t, h.ctx, func() (bool, error) {
		return !bookmarkExists(h.t, h.Repo(), h.Env(), "first-bm") && !bookmarkExists(h.t, h.Repo(), h.Env(), "second-bm"), nil
	})

	h.Quit()
}
