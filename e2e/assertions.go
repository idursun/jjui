package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func countRevisions(t *testing.T, repo string, env []string) int {
	t.Helper()
	output := runCommand(t, repo, env, "jj", "log", "-r", "all()", "-T", "change_id ++ \"\\n\"")
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func workingCopyDescription(t *testing.T, repo string, env []string) string {
	t.Helper()
	return strings.TrimSpace(runCommand(t, repo, env, "jj", "log", "-r", "@", "-T", "description", "--no-graph", "--color", "never", "--quiet"))
}

func bookmarkExists(t *testing.T, repo string, env []string, bookmark string) bool {
	t.Helper()
	output := runCommand(t, repo, env, "jj", "bookmark", "list", "--color", "never")
	return strings.Contains(output, bookmark)
}

func waitFor(t *testing.T, ctx context.Context, condition func() (bool, error)) {
	t.Helper()
	waitCtx, cancel := context.WithTimeout(ctx, perWaitTimeout)
	defer cancel()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		ok, err := condition()
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			return
		}
		select {
		case <-waitCtx.Done():
			t.Fatal(waitCtx.Err())
		case <-ticker.C:
		}
	}
}

func screenContains(screen []string, want string) bool {
	for _, row := range screen {
		if strings.Contains(row, want) {
			return true
		}
	}
	return false
}

func screenContainsCheckmark(screen []string) bool {
	return screenContains(screen, "✓")
}

func screenRowIndexContaining(screen []string, want string) int {
	for index, row := range screen {
		if strings.Contains(row, want) {
			return index
		}
	}
	return -1
}

func logFailureDiagnostics(t *testing.T, session *ptySession, repo string, env []string) {
	t.Helper()

	if screen, err := session.Snapshot(); err != nil {
		t.Logf("screen capture unavailable: %v", err)
	} else {
		t.Logf("screen capture:\n%s", formatScreen(screen))
	}
	if styled, err := session.StyledSnapshot(); err != nil {
		t.Logf("styled screen capture unavailable: %v", err)
	} else {
		t.Logf("styled screen capture:\n%s", formatStyledScreen(styled))
	}
	trace := session.Trace()
	if len(trace) > 0 {
		var events strings.Builder
		for _, event := range trace {
			fmt.Fprintf(&events, "+%s\t%s\n", event.at.Round(time.Millisecond), event.message)
		}
		t.Logf("action trace:\n%s", events.String())
	}
	t.Logf("raw PTY output:\n%s", tailString(session.RawOutput(), 64*1024))

	diagnostics := []struct {
		name string
		args []string
	}{
		{name: "jj status", args: []string{"status", "--no-pager"}},
		{name: "jj log", args: []string{"log", "--no-pager", "-r", "all()", "--color", "never"}},
		{name: "jj op log", args: []string{"op", "log", "--no-pager", "-n", "20", "--color", "never"}},
	}

	for _, diagnostic := range diagnostics {
		output, err := commandOutput(repo, env, "jj", diagnostic.args...)
		if err != nil {
			t.Logf("%s failed: %v\n%s", diagnostic.name, err, output)
			continue
		}
		t.Logf("%s:\n%s", diagnostic.name, output)
	}
}
