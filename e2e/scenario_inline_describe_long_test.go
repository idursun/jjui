//go:build e2e

package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

const longDescriptionLines = 40

func longDescriptionLine(n int) string {
	return fmt.Sprintf("desc line %02d", n)
}

// startLongInlineDescribe opens inline describe on a revision whose description
// is taller than the 30-row terminal and which is the last row of the revset.
func startLongInlineDescribe(t *testing.T) (*ptySession, context.Context) {
	t.Helper()
	jjuiBin := jjuiBinary(t)
	repo := newTestRepo(t)
	lines := make([]string, longDescriptionLines)
	for i := range lines {
		lines[i] = longDescriptionLine(i + 1)
	}
	repo.JJ("describe", "-m", strings.Join(lines, "\n"))
	writeJJUIConfig(t, repo.Env(), "[revisions]\nrevset = \"@\"\n")

	session, ctx := startJJUITestWithRepo(t, jjuiBin, repo, longDescriptionLine(1))
	if err := session.SendKey(ghostty.KeyEnter, "", 0); err != nil {
		t.Fatal(err)
	}
	// The log only shows the first description line, so a second one means the editor is open.
	if _, err := session.WaitForStableScreen(ctx, 3, func(screen []string) bool {
		visible := 0
		for i := 1; i <= longDescriptionLines; i++ {
			if screenContains(screen, longDescriptionLine(i)) {
				visible++
			}
		}
		return visible >= 2
	}); err != nil {
		t.Fatalf("inline describe did not open: %v\nraw output:\n%s", err, session.RawOutput())
	}
	return session, ctx
}

func waitStable(t *testing.T, session *ptySession, ctx context.Context) []string {
	t.Helper()
	screen, err := session.WaitForStableScreen(ctx, 3, func([]string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	return screen
}

// waitCursorOnLine waits until the line being edited is on screen and the
// terminal cursor sits on it. Moving the cursor inside the visible part of the
// editor leaves the screen text unchanged, so a stable screen is not enough.
func waitCursorOnLine(t *testing.T, session *ptySession, ctx context.Context, line int) []string {
	t.Helper()
	want := longDescriptionLine(line)
	var screen []string
	_, err := session.WaitForCursor(ctx, func(cursor cursorPosition) bool {
		snapshot, err := session.Snapshot()
		if err != nil {
			return false
		}
		screen = snapshot
		row := screenRowIndexContaining(screen, want)
		return row >= 0 && cursor.Visible && cursor.InViewport && int(cursor.Y) == row
	})
	if err != nil {
		t.Fatalf("cursor is not on %q: %v\nscreen:\n%s", want, err, formatScreen(screen))
	}
	return screen
}

func sendWheelDown(t *testing.T, session *ptySession, x, y int) {
	t.Helper()
	// SGR mouse encoding, 1-based coordinates; button 65 is wheel down.
	if err := session.SendText(fmt.Sprintf("\x1b[<65;%d;%dM", x, y)); err != nil {
		t.Fatal(err)
	}
}

// Opening inline describe should show the cursor (end of the description)
// on the first frame, not the start of the description.
func Test_Revisions_InlineDescribeLongDescriptionOpensAtCursor(t *testing.T) {
	t.Parallel()
	session, ctx := startLongInlineDescribe(t)

	waitCursorOnLine(t, session, ctx, longDescriptionLines)
}

// Moving the cursor within the visible part of the editor must not scroll it.
func Test_Revisions_InlineDescribeLongDescriptionKeepsScrollPosition(t *testing.T) {
	t.Parallel()
	session, ctx := startLongInlineDescribe(t)

	// Any key forces a re-render, which works around the first-frame bug.
	if err := session.SendKey(ghostty.KeyArrowDown, "", 0); err != nil {
		t.Fatal(err)
	}
	before := waitCursorOnLine(t, session, ctx, longDescriptionLines)
	lastRow := screenRowIndexContaining(before, longDescriptionLine(longDescriptionLines))

	if err := session.SendKey(ghostty.KeyArrowUp, "", 0); err != nil {
		t.Fatal(err)
	}
	after := waitCursorOnLine(t, session, ctx, longDescriptionLines-1)
	if row := screenRowIndexContaining(after, longDescriptionLine(longDescriptionLines)); row != lastRow {
		t.Fatalf("%q moved from row %d to %d; the view should not scroll\nbefore:\n%s\nafter:\n%s",
			longDescriptionLine(longDescriptionLines), lastRow, row, formatScreen(before), formatScreen(after))
	}
}

// Wheel-scrolling the list must not detach the editor from its cursor.
func Test_Revisions_InlineDescribeLongDescriptionFollowsCursorAfterWheel(t *testing.T) {
	t.Parallel()
	session, ctx := startLongInlineDescribe(t)

	if err := session.SendKey(ghostty.KeyArrowDown, "", 0); err != nil {
		t.Fatal(err)
	}
	waitStable(t, session, ctx)

	sendWheelDown(t, session, 10, 10)
	sendWheelDown(t, session, 10, 10)
	waitStable(t, session, ctx)

	const ups = 5
	for range ups {
		if err := session.SendKey(ghostty.KeyArrowUp, "", 0); err != nil {
			t.Fatal(err)
		}
	}
	waitCursorOnLine(t, session, ctx, longDescriptionLines-ups)
}
