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

func bookmarkTarget(t *testing.T, repo *testRepo, name string) string {
	return strings.TrimSpace(runCommand(t, repo.Path(), repo.Env(), "jj", "log", "--ignore-working-copy", "--no-graph", "--color", "never", "-r", name, "-T", "change_id"))
}

func Test_TextObjects_RenamesFocusedBookmark(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	repo.JJ("bookmark", "create", "old-name", "-r", "@")

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "old-name")
	focusTextOnRow(t, session, ctx, "old-name")
	if err := session.SendKey(ghostty.KeyR, "r", 0); err != nil {
		t.Fatal(err)
	}
	// The inline editor shows the current name next to the bookmark itself.
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		row := screenRowIndexContaining(screen, "old-name")
		return row >= 0 && strings.Count(screen[row], "old-name") == 2
	}); err != nil {
		t.Fatalf("inline rename did not open: %v", err)
	}
	if err := session.SendKey(ghostty.KeyU, "u", ghostty.ModCtrl); err != nil {
		t.Fatal(err)
	}
	if err := session.SendText("new-name"); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		return bookmarkExists(t, repo.Path(), repo.Env(), "new-name") && !bookmarkExists(t, repo.Path(), repo.Env(), "old-name"), nil
	})
}

func Test_TextObjects_MovesFocusedBookmark(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	repo.JJ("bookmark", "create", "mover", "-r", "@-")
	workingCopy := bookmarkTarget(t, repo, "@")

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "mover")
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	focusTextOnRow(t, session, ctx, "mover")
	if err := session.SendKey(ghostty.KeyM, "m", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool { return screenContains(screen, "<< onto >>") }); err != nil {
		t.Fatalf("move bookmark did not start: %v", err)
	}
	if err := session.SendKey(ghostty.KeyK, "k", 0); err != nil {
		t.Fatal(err)
	}
	// Wait until the target follows the cursor before applying.
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		marker, workingCopy := screenRowIndexContaining(screen, "<< onto >>"), screenRowIndexContaining(screen, "(empty)")
		return marker >= 0 && workingCopy >= 0 && marker < workingCopy
	}); err != nil {
		t.Fatalf("move target did not follow the cursor: %v", err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		return bookmarkTarget(t, repo, "mover") == workingCopy, nil
	})
}
