//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Theme_AppliesConfiguredTheme(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	writeJJUITheme(t, repo.Env(), "contrast", `
[colors]
"status title" = { fg = "#123456", bg = "#654321", bold = true }
`)
	writeJJUIConfig(t, repo.Env(), `
[ui]
theme = "contrast"
`)

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "initial")
	wantForeground := ghostty.ColorRGB{R: 0x12, G: 0x34, B: 0x56}
	wantBackground := ghostty.ColorRGB{R: 0x65, G: 0x43, B: 0x21}
	_, err := session.WaitForStyledScreen(ctx, func(screen styledScreen) bool {
		return screen.hasStyle(func(cell screenCell) bool {
			return cell.Style.HasForeground && cell.Style.Foreground == wantForeground &&
				cell.Style.HasBackground && cell.Style.Background == wantBackground &&
				cell.Style.Bold
		})
	})
	if err != nil {
		t.Fatalf("configured theme was not applied to the rendered screen: %v", err)
	}

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
