//go:build e2e

package main

import (
	"strings"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Revset_EditingPositionsCursorAndHandlesInsertion(t *testing.T) {
	t.Parallel()
	_, session, ctx := startJJUITest(t, "initial")
	expression := "builtin_log()"

	if err := session.SendKey(ghostty.KeyL, "L", ghostty.ModShift); err != nil {
		t.Fatal(err)
	}
	screen, err := session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "revset:") && screenContains(screen, "apply")
	})
	if err != nil {
		t.Fatalf("revset editor did not render its input: %v", err)
	}
	row := screenRowIndexContaining(screen, "revset:")
	column := strings.Index(screen[row], "revset:") + len("revset: ")
	if _, err := session.WaitForCursor(ctx, func(cursor cursorPosition) bool {
		return cursor.Visible && cursor.InViewport && cursor.X == uint16(column) && cursor.Y == uint16(row)
	}); err != nil {
		t.Fatalf("revset cursor was not placed at the start of the input: %v", err)
	}

	if err := session.SendText(expression); err != nil {
		t.Fatal(err)
	}
	screen, err = session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "revset: "+expression)
	})
	if err != nil {
		t.Fatalf("revset editor did not render the typed expression: %v", err)
	}
	row = screenRowIndexContaining(screen, "revset: "+expression)
	column = strings.Index(screen[row], expression) + len(expression)
	if _, err := session.WaitForCursor(ctx, func(cursor cursorPosition) bool {
		return cursor.Visible && cursor.InViewport && cursor.X == uint16(column) && cursor.Y == uint16(row)
	}); err != nil {
		t.Fatalf("revset cursor was not placed at the end of the typed expression: %v", err)
	}

	if err := session.SendKey(ghostty.KeyArrowLeft, "", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyArrowLeft, "", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendText("X"); err != nil {
		t.Fatal(err)
	}
	expression = "builtin_logX()"
	screen, err = session.WaitForScreen(ctx, func(screen []string) bool {
		return screenContains(screen, "revset: "+expression)
	})
	if err != nil {
		t.Fatalf("revset editor did not insert text at the cursor: %v", err)
	}
	row = screenRowIndexContaining(screen, "revset: "+expression)
	column = strings.Index(screen[row], expression) + len("builtin_logX")
	if _, err := session.WaitForCursor(ctx, func(cursor cursorPosition) bool {
		return cursor.Visible && cursor.InViewport && cursor.X == uint16(column) && cursor.Y == uint16(row)
	}); err != nil {
		t.Fatalf("revset cursor did not follow the inserted text: %v", err)
	}

	if err := session.SendKey(ghostty.KeyEscape, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := session.WaitForStableScreen(ctx, 3, func(screen []string) bool {
		return screenContains(screen, "initial") && !screenContains(screen, "apply")
	}); err != nil {
		t.Fatalf("escape did not close the revset editor: %v", err)
	}
	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
