//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Revisions_NewRevisionUpdatesRepository(t *testing.T) {
	t.Parallel()
	repo, session, ctx := startJJUITest(t, "initial")
	before := countRevisions(t, repo.Path(), repo.Env())

	// 'n' is jjui's default "new revision" action. The PTY receives the
	// encoded key, jjui runs `jj new`, and the repository should gain one
	// additional revision.
	if err := session.SendKey(ghostty.KeyN, "n", 0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, ctx, func() (bool, error) {
		count := countRevisions(t, repo.Path(), repo.Env())
		return count == before+1, nil
	})

	// Quit through the same PTY path so the child restores its terminal and
	// exits normally.
	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v\nraw output:\n%s", err, session.RawOutput())
	}

	if got := countRevisions(t, repo.Path(), repo.Env()); got != before+1 {
		t.Fatalf("revision count after jjui scenario = %d, want %d", got, before+1)
	}
}
