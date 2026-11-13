//go:build e2e

package main

import (
	"context"
	"testing"
	"time"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_PTY_AnswersTerminalQueries(t *testing.T) {
	t.Parallel()
	term, err := ghostty.NewTerminal(ghostty.WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(term.Close)

	// A real child queries its cursor position and checks the terminal's reply.
	// Without the write-PTY callback, it blocks waiting for input.
	session, err := startPTY(term, []string{"sh", "-c", `
        stty raw -echo
        printf '\033[6n'
        reply=$(dd bs=1 count=6 2>/dev/null)
        test "$reply" = "$(printf '\033[1;1R')"
    `}, t.TempDir(), nil, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := session.ExpectExit(ctx, 0); err != nil {
		t.Fatal(err)
	}
}
