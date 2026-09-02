//go:build e2e

package main

import (
	"path/filepath"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Bookmarks_ShowsTypedName(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	configHome := environmentValue(repo.Env(), "XDG_CONFIG_HOME")
	appendToFile(t, filepath.Join(configHome, "jj", "config.toml"), `
[template-aliases]
'format_short_id(id)' = 'id.shortest()'
'format_short_commit_id(id)' = 'id.short(8)'
`)

	jjuiBin := jjuiBinary(t)
	session, ctx := startJJUITestWithRepo(t, jjuiBin, repo, "initial")
	if err := session.SendKey(ghostty.KeyB, "B", ghostty.ModShift); err != nil {
		t.Fatal(err)
	}
	const bookmark = "visible-bookmark"
	if err := session.SendText(bookmark); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, bookmark)
	}); err != nil {
		t.Fatalf("typed bookmark name was not rendered: %v", err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}

	waitFor(t, ctx, func() (bool, error) {
		return bookmarkExists(t, repo.Path(), repo.Env(), bookmark), nil
	})
	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
