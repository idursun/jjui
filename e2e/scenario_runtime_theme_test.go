//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Theme_ChangesAtRuntime(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	writeJJUITheme(t, repo.Env(), "first", `
[colors]
"status title" = { fg = "#102030", bg = "#304050", bold = true }
`)
	writeJJUITheme(t, repo.Env(), "second", `
[colors]
"status title" = { fg = "#d0e0f0", bg = "#f0e0d0", underline = true }
`)
	writeJJUIConfig(t, repo.Env(), `
[ui]
theme = "first"

[[actions]]
name = "switch-theme"
lua = '''
set_theme("second")
'''

[[bindings]]
action = "switch-theme"
key = "Y"
scope = "revisions"
desc = "switch theme"
`)

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "initial")
	firstForeground := ghostty.ColorRGB{R: 0x10, G: 0x20, B: 0x30}
	firstBackground := ghostty.ColorRGB{R: 0x30, G: 0x40, B: 0x50}
	if _, err := session.WaitForStyledScreen(ctx, func(screen styledScreen) bool {
		return screen.hasStyle(func(cell screenCell) bool {
			return cell.Style.HasForeground && cell.Style.Foreground == firstForeground &&
				cell.Style.HasBackground && cell.Style.Background == firstBackground &&
				cell.Style.Bold
		})
	}); err != nil {
		t.Fatalf("initial runtime theme was not rendered: %v", err)
	}

	if err := session.SendKey(ghostty.KeyY, "Y", 0); err != nil {
		t.Fatal(err)
	}
	secondForeground := ghostty.ColorRGB{R: 0xd0, G: 0xe0, B: 0xf0}
	secondBackground := ghostty.ColorRGB{R: 0xf0, G: 0xe0, B: 0xd0}
	if _, err := session.WaitForStyledScreen(ctx, func(screen styledScreen) bool {
		return screen.hasStyle(func(cell screenCell) bool {
			return cell.Style.HasForeground && cell.Style.Foreground == secondForeground &&
				cell.Style.HasBackground && cell.Style.Background == secondBackground &&
				cell.Style.Underline && !cell.Style.Bold
		})
	}); err != nil {
		t.Fatalf("runtime theme change was not rendered: %v", err)
	}

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
