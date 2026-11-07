//go:build e2e

package main

import "testing"

func Test_StatusHelpDoesNotShowDormantRevsetScopeInStackedView(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.WriteConfigTOML(`
[[bindings]]
action = "ui.open_git"
key = "Y"
scope = "revisions"
desc = "git"
`)
	h.Start("initial")

	h.Key("Y")
	h.WaitText("Git Operations")
	h.Text("?")
	if _, err := h.session.WaitForStableScreen(h.ctx, 3, func(screen []string) bool {
		return screenContains(screen, "Git") &&
			statusHasEntry(screen, "tab", "next remote") &&
			!screenContains(screen, "Revset") &&
			!statusHasEntry(screen, "tab", "autocomplete")
	}); err != nil {
		screen, snapshotErr := h.session.Snapshot()
		if snapshotErr != nil {
			t.Fatalf("could not inspect expanded status help: %v", snapshotErr)
		}
		t.Fatalf("expanded git help did not show git actions or included dormant revset help:\n%s", formatScreen(screen))
	}

	h.Key("Escape")
	h.Quit()
}
