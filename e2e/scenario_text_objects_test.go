package main

import (
	"context"
	"path/filepath"
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
	// The name is edited in place: the cursor moves to the end of it.
	screen, err := session.WaitForScreen(ctx, func(screen []string) bool { return screenContains(screen, "old-name") })
	if err != nil {
		t.Fatal(err)
	}
	row := screenRowIndexContaining(screen, "old-name")
	end := len([]rune(screen[row][:strings.Index(screen[row], "old-name")])) + len("old-name")
	if _, err := session.WaitForCursor(ctx, func(c cursorPosition) bool { return int(c.Y) == row && int(c.X) == end }); err != nil {
		t.Fatalf("rename did not start editing the name in place: %v", err)
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

// newRepoWithFileRemote returns a repository with a bare Git repository as
// its origin remote.
func newRepoWithFileRemote(t *testing.T) (*testRepo, string) {
	repo := newTestRepo(t)
	root := filepath.Dir(repo.Path())
	remote := filepath.Join(root, "remote.git")
	runCommand(t, root, repo.Env(), "git", "init", "--bare", remote)
	repo.JJ("git", "remote", "add", "origin", remote)
	return repo, remote
}

func remoteHasBranch(t *testing.T, repo *testRepo, remote string, branch string) bool {
	_, err := commandOutput(filepath.Dir(repo.Path()), repo.Env(), "git", "--git-dir", remote, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

func Test_TextObjects_DeletesTrackedBookmarkOnRemote(t *testing.T) {
	t.Parallel()
	repo, remote := newRepoWithFileRemote(t)
	repo.JJ("describe", "-m", "to delete")
	repo.JJ("bookmark", "create", "gone", "-r", "@")
	repo.JJ("bookmark", "track", "gone", "--remote", "origin")
	repo.JJ("git", "push", "--bookmark", "gone")
	if !remoteHasBranch(t, repo, remote, "gone") {
		t.Fatal("setup did not push the bookmark")
	}

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "gone")
	focusTextOnRow(t, session, ctx, "gone")
	if err := session.SendKey(ghostty.KeyD, "d", 0); err != nil {
		t.Fatal(err)
	}
	// Deleting the local bookmark leaves the remote one to push.
	focusTextOnRow(t, session, ctx, "gone@origin")
	if err := session.SendKey(ghostty.KeyG, "G", ghostty.ModShift); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		return !remoteHasBranch(t, repo, remote, "gone"), nil
	})
}

func Test_TextObjects_DeletesUntrackedRemoteBookmark(t *testing.T) {
	t.Parallel()
	repo, remote := newRepoWithFileRemote(t)
	// On @- so that the working copy is not the commit that turns immutable.
	repo.JJ("bookmark", "create", "other", "-r", "@-")
	repo.JJ("bookmark", "track", "other", "--remote", "origin")
	repo.JJ("git", "push", "--bookmark", "other")
	// Leave only the remote bookmark, untracked.
	repo.JJ("bookmark", "untrack", "other", "--remote", "origin")
	repo.JJ("bookmark", "delete", "other")
	if !remoteHasBranch(t, repo, remote, "other") {
		t.Fatal("setup did not push the bookmark")
	}

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "other@origin")
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	focusTextOnRow(t, session, ctx, "other@origin")
	if err := session.SendKey(ghostty.KeyD, "d", 0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		out := runCommand(t, repo.Path(), repo.Env(), "jj", "bookmark", "list", "--ignore-working-copy", "other")
		return strings.Contains(out, "(deleted)"), nil
	})
	focusTextOnRow(t, session, ctx, "other@origin")
	if err := session.SendKey(ghostty.KeyG, "G", ghostty.ModShift); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		return !remoteHasBranch(t, repo, remote, "other"), nil
	})
}

func Test_TextObjects_DeletesFocusedTag(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	// Tagged commits are immutable, so tag @- to keep the working copy put.
	repo.JJ("tag", "set", "v1.0", "-r", "@-")

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "v1.0")
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	focusTextOnRow(t, session, ctx, "v1.0")
	if err := session.SendKey(ghostty.KeyD, "d", 0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		out := runCommand(t, repo.Path(), repo.Env(), "jj", "tag", "list", "--ignore-working-copy")
		return !strings.Contains(out, "v1.0"), nil
	})
}

func Test_TextObjects_UpdatesFocusedChangeId(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	changeId := func() string {
		return strings.TrimSpace(runCommand(t, repo.Path(), repo.Env(), "jj", "log", "--ignore-working-copy", "-r", "@", "--no-graph", "--color", "never", "-T", "change_id.short(8)"))
	}
	before := changeId()

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, before)
	focusTextOnRow(t, session, ctx, before)
	if err := session.SendKey(ghostty.KeyR, "r", 0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		return changeId() != before, nil
	})
}

func Test_TextObjects_ForgetsFocusedBookmark(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	repo.JJ("bookmark", "create", "forget-me", "-r", "@")

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "forget-me")
	focusTextOnRow(t, session, ctx, "forget-me")
	if err := session.SendKey(ghostty.KeyD, "D", ghostty.ModShift); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		return !bookmarkExists(t, repo.Path(), repo.Env(), "forget-me"), nil
	})
}

func Test_TextObjects_PushesFocusedChange(t *testing.T) {
	t.Parallel()
	repo, remote := newRepoWithFileRemote(t)
	repo.JJ("describe", "-m", "push me")
	changeId := strings.TrimSpace(runCommand(t, repo.Path(), repo.Env(), "jj", "log", "--ignore-working-copy", "-r", "@", "--no-graph", "--color", "never", "-T", "change_id.short(8)"))

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, changeId)
	focusTextOnRow(t, session, ctx, changeId)
	if err := session.SendKey(ghostty.KeyG, "G", ghostty.ModShift); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		out, _ := commandOutput(filepath.Dir(repo.Path()), repo.Env(), "git", "--git-dir", remote, "for-each-ref", "--format=%(refname)", "refs/heads/")
		return strings.Contains(out, "refs/heads/push-"+changeId), nil
	})
}

func Test_TextObjects_ForgetsTrackedBookmarkWithItsRemotes(t *testing.T) {
	t.Parallel()
	repo, _ := newRepoWithFileRemote(t)
	repo.JJ("bookmark", "create", "forget-me", "-r", "@-")
	repo.JJ("bookmark", "track", "forget-me", "--remote", "origin")
	repo.JJ("git", "push", "--bookmark", "forget-me")

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "forget-me")
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	focusTextOnRow(t, session, ctx, "forget-me")
	if err := session.SendKey(ghostty.KeyD, "D", ghostty.ModShift); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		out := runCommand(t, repo.Path(), repo.Env(), "jj", "bookmark", "list", "--all-remotes", "--color", "never")
		return !strings.Contains(out, "forget-me"), nil
	})
}
