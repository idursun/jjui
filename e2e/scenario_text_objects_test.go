package main

import (
	"context"
	"strings"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

// focusTextOnRow presses ] until the terminal cursor sits on the first
// occurrence of text on the screen.
func focusTextOnRow(t *testing.T, session *ptySession, ctx context.Context, text string) {
	t.Helper()
	screen, err := session.WaitForScreen(ctx, func(screen []string) bool { return screenContains(screen, text) })
	if err != nil {
		t.Fatalf("%q not on screen: %v", text, err)
	}
	wantY, wantX := -1, -1
	for y, line := range screen {
		if i := strings.Index(line, text); i >= 0 {
			wantY, wantX = y, len([]rune(line[:i]))
			break
		}
	}
	for range 10 {
		cursor, err := session.Cursor()
		if err != nil {
			t.Fatal(err)
		}
		if int(cursor.Y) == wantY && int(cursor.X) == wantX {
			return
		}
		if err := session.SendKey(ghostty.KeyBracketRight, "]", 0); err != nil {
			t.Fatal(err)
		}
		_, _ = session.WaitForCursor(ctx, func(c cursorPosition) bool { return c != cursor })
	}
	cursor, _ := session.Cursor()
	t.Fatalf("cursor never reached %q at (%d,%d); last at (%d,%d)\n%s", text, wantX, wantY, cursor.X, cursor.Y, strings.Join(screen, "\n"))
}

func Test_TextObjects_DeletesFocusedBookmark(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	repo.JJ("describe", "-m", "bookmarked")
	repo.JJ("bookmark", "create", "focus-me", "-r", "@")

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "focus-me")
	focusTextOnRow(t, session, ctx, "focus-me")
	if err := session.SendKey(ghostty.KeyD, "d", 0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		return !bookmarkExists(t, repo.Path(), repo.Env(), "focus-me"), nil
	})
}

func Test_TextObjects_ResetsFocusedAuthor(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	repo.JJ("describe", "-m", "borrowed")
	repo.JJ("metaedit", "--author", "Other <other@example.com>")

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "other@example.com")
	focusTextOnRow(t, session, ctx, "other@example.com")
	if err := session.SendKey(ghostty.KeyR, "r", 0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		author := runCommand(t, repo.Path(), repo.Env(), "jj", "log", "--ignore-working-copy", "-r", "@", "--no-graph", "--color", "never", "-T", "author.email()")
		return strings.TrimSpace(author) == "pty@example.com", nil
	})
}
