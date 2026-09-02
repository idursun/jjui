//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Revisions_SelectionClearsWithEscape(t *testing.T) {
	t.Parallel()
	_, session, ctx := startJJUITest(t, "initial")

	if err := session.SendKey(ghostty.KeySpace, " ", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForScreen(ctx, screenContainsCheckmark); err != nil {
		t.Fatalf("space did not select the focused revision: %v\nraw output:\n%s", err, session.RawOutput())
	}

	if err := session.SendKey(ghostty.KeyEscape, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForStableScreen(ctx, 3, func(screen []string) bool {
		return screenContains(screen, "initial") && !screenContainsCheckmark(screen)
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
