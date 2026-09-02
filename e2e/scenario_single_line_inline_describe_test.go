//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Revisions_InlineDescribeRendersOnSingleLineLog(t *testing.T) {
	t.Parallel()
	jjuiBin := jjuiBinary(t)

	repo := newTestRepo(t)
	repo.JJ("config", "set", "--repo", "templates.log", "builtin_log_oneline")
	session, ctx := startJJUITestWithRepo(t, jjuiBin, repo, "initial")

	// Select the committed revision. The working-copy revision is selected
	// initially and is rendered separately from the single-line target.
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendText(" inline description"); err != nil {
		t.Fatal(err)
	}

	screen, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "initial") && screenContains(screen, "inline description")
	})
	if err != nil {
		t.Fatalf("inline describe was not rendered on the single-line revision: %v\nraw output:\n%s", err, session.RawOutput())
	}
	initialRow := screenRowIndexContaining(screen, "initial")
	overlayRow := screenRowIndexContaining(screen, "inline description")
	if overlayRow != initialRow+1 {
		t.Fatalf("inline description row = %d, initial revision row = %d; want overlay immediately after revision\nscreen:\n%s", overlayRow, initialRow, formatScreen(screen))
	}

	if err := session.SendKey(ghostty.KeyEscape, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForStableScreen(ctx, 3, func(screen []string) bool {
		return screenContains(screen, "initial") && !screenContains(screen, "inline description")
	}); err != nil {
		t.Fatalf("escape did not close inline describe: %v\nraw output:\n%s", err, session.RawOutput())
	}
	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v\nraw output:\n%s", err, session.RawOutput())
	}
}
