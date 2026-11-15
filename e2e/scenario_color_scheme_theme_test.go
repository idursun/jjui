//go:build e2e

package main

import (
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Theme_FollowsTerminalColorScheme(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	writeJJUITheme(t, repo.Env(), "night", `
[colors]
"status title" = { fg = "#102030", bg = "#304050", bold = true }
`)
	writeJJUITheme(t, repo.Env(), "day", `
[colors]
"status title" = { fg = "#d0e0f0", bg = "#f0e0d0", underline = true }
`)
	writeJJUIConfig(t, repo.Env(), `
[ui.theme]
dark = "night"
light = "day"
`)

	session, ctx := startJJUITestWithRepo(t, jjuiBinary(t), repo, "initial")

	waitForTitle := func(fg, bg ghostty.ColorRGB, scheme string) {
		t.Helper()
		if _, err := session.WaitForStyledScreen(ctx, func(screen styledScreen) bool {
			return screen.hasStyle(func(cell screenCell) bool {
				return cell.Style.HasForeground && cell.Style.Foreground == fg &&
					cell.Style.HasBackground && cell.Style.Background == bg
			})
		}); err != nil {
			t.Fatalf("%s theme was not rendered: %v", scheme, err)
		}
	}
	sendScheme := func(scheme ghostty.ColorScheme) {
		t.Helper()
		report, err := ghostty.ColorSchemeReportEncode(scheme)
		if err != nil {
			t.Fatal(err)
		}
		if err := session.SendText(string(report)); err != nil {
			t.Fatal(err)
		}
	}

	dayFg := ghostty.ColorRGB{R: 0xd0, G: 0xe0, B: 0xf0}
	dayBg := ghostty.ColorRGB{R: 0xf0, G: 0xe0, B: 0xd0}
	nightFg := ghostty.ColorRGB{R: 0x10, G: 0x20, B: 0x30}
	nightBg := ghostty.ColorRGB{R: 0x30, G: 0x40, B: 0x50}

	sendScheme(ghostty.ColorSchemeLight)
	waitForTitle(dayFg, dayBg, "light")

	sendScheme(ghostty.ColorSchemeDark)
	waitForTitle(nightFg, nightBg, "dark")

	if err := session.SendKey(ghostty.KeyQ, "q", 0); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitContext(ctx); err != nil {
		t.Fatalf("jjui exited with error: %v", err)
	}
}
