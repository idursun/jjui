//go:build e2e

package main

import (
	"strings"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Preview_AutoPlacementFollowsTerminalSize(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t).
		Append("README.md", "auto-preview-marker\n").
		Commit("auto-preview")
	writeJJUIConfig(t, repo.Env(), `
[preview]
position = "auto"
`)

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "auto-preview")
	if err := session.SendKey(ghostty.KeyJ, "j", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.SendKey(ghostty.KeyP, "p", 0); err != nil {
		t.Fatal(err)
	}

	screen, err := session.WaitForStableScreen(ctx, 3, func(screen []string) bool {
		return screenContains(screen, "auto-preview-marker") && hasVerticalPreviewSeparator(screen)
	})
	if err != nil {
		t.Fatalf("auto preview did not open on the right at 100x30: %v", err)
	}
	if hasHorizontalPreviewSeparator(screen) {
		t.Fatalf("auto preview unexpectedly opened at the bottom at 100x30:\n%s", formatScreen(screen))
	}

	if err := session.Resize(40, 80); err != nil {
		t.Fatalf("resize to 40x80: %v", err)
	}
	screen, err = session.WaitForStableScreen(ctx, 3, func(screen []string) bool {
		return screenContains(screen, "auto-preview-marker") && hasHorizontalPreviewSeparator(screen)
	})
	if err != nil {
		t.Fatalf("auto preview did not move to the bottom at 40x80: %v", err)
	}
	if hasVerticalPreviewSeparator(screen) {
		t.Fatalf("auto preview unexpectedly remained on the right at 40x80:\n%s", formatScreen(screen))
	}

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}

func hasVerticalPreviewSeparator(screen []string) bool {
	maxCount := 0
	for column := 1; column < screenWidth(screen)-1; column++ {
		count := 0
		for _, row := range screen {
			if runeAt(row, column) == '│' {
				count++
			}
		}
		if count > maxCount {
			maxCount = count
		}
	}
	return maxCount >= 10
}

func hasHorizontalPreviewSeparator(screen []string) bool {
	for _, row := range screen {
		if strings.Count(row, "─") >= screenWidth(screen)/2 {
			return true
		}
	}
	return false
}

func screenWidth(screen []string) int {
	if len(screen) == 0 {
		return 0
	}
	return len([]rune(screen[0]))
}

func runeAt(row string, column int) rune {
	runes := []rune(row)
	if column < 0 || column >= len(runes) {
		return 0
	}
	return runes[column]
}
