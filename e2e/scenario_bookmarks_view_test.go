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

func startBookmarksView(t *testing.T) *Harness {
	t.Helper()
	h := NewHarness(t)
	h.repo.Write("a.txt", "bookmarks view\n").Commit("bookmarks view target")
	h.repo.JJ("bookmark", "create", "esc-feature", "-r", "@-")
	h.WriteConfigLua(`
function setup(config)
  config.action("bookmarks-view-flash", function()
    flash({text = "bookmarks-view-flash-message", error = true})
  end, { key = "Y", scope = "revisions" })
end
`)
	h.Start("bookmarks view target")
	h.Key("b")
	h.WaitText("filter:")
	// The rows have loaded once the bookmark object is focused.
	h.WaitMode("bookmark")
	return h
}

// Esc closes what is drawn on top of the bookmarks view before the view itself.
func Test_BookmarksView_EscClosesFlashFirst(t *testing.T) {
	t.Parallel()
	h := startBookmarksView(t)

	h.Key("Y")
	h.WaitText("bookmarks-view-flash-message")
	h.Key("Escape")
	h.WaitNoText("bookmarks-view-flash-message")
	h.WaitText("filter:")

	h.Key("Escape")
	h.WaitNoText("filter:")
	h.Quit()
}

func Test_BookmarksView_EscClosesExpandedHelpFirst(t *testing.T) {
	t.Parallel()
	h := startBookmarksView(t)

	h.Text("?")
	h.WaitText("Revisions")
	h.Key("Escape")
	h.WaitNoText("Revisions")
	h.WaitText("filter:")

	h.Key("Escape")
	h.WaitNoText("filter:")
	h.Quit()
}

// The focused object's actions lead the status bar; the bookmarks view's keys
// follow and still work.
func Test_BookmarksView_StatusLeadsWithFocusedObject(t *testing.T) {
	t.Parallel()
	h := startBookmarksView(t)

	screen := h.WaitText("rename")
	if !statusHasEntry(screen[len(screen)-1:], "r", "rename") {
		t.Fatalf("status bar did not lead with the bookmark actions:\n%s", formatScreen(screen))
	}

	if err := h.session.SendKey(ghostty.KeyTab, "\t", 0); err != nil {
		t.Fatal(err)
	}
	h.WaitNoText("remote_bookmarks()")
	h.WaitText("revset: bookmarks()")

	h.Key("k")
	h.WaitMode("remotes")
	h.Quit()
}

func Test_BookmarksView_PushesDeletionsToTheFocusedRemote(t *testing.T) {
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
	send(ghostty.KeyB, "b")
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool { return screenContains(screen, "gone@origin") }); err != nil {
		t.Fatalf("the bookmarks view did not open: %v", err)
	}
	// k on the top revision focuses the remotes; l moves from all to local
	// to origin.
	send(ghostty.KeyK, "k")
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool { return screenContains(screen, "back to log") }); err != nil {
		t.Fatalf("k did not focus the remotes: %v", err)
	}
	// Two quick presses: the log must follow the second one.
	send(ghostty.KeyL, "l")
	send(ghostty.KeyL, "l")
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool { return screenContains(screen, `remote=exact:"origin"`) }); err != nil {
		t.Fatalf("l did not move to origin: %v", err)
	}
	if err := session.SendKey(ghostty.KeyD, "d", 0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		return !remoteHasBranch(t, repo, remote, "gone"), nil
	})
}
