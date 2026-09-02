//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Revisions_InlineDescribeAcceptsLeadingDash(t *testing.T) {
	t.Parallel()
	repo, session, ctx := startJJUITest(t, "initial")
	const description = "-foo"

	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendText(description); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyEnter, "", ghostty.ModAlt); err != nil {
		t.Fatal(err)
	}

	waitFor(t, ctx, func() (bool, error) {
		return workingCopyDescription(t, repo.Path(), repo.Env()) == description, nil
	})

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v\nraw output:\n%s", err, session.RawOutput())
	}
}
