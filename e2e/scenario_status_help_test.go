//go:build e2e

package main

import "testing"

func Test_StatusHelpDoesNotShowDormantRevsetScopeInStackedView(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.WriteConfigTOML(`
[[bindings]]
action = "ui.open_undo"
key = "Y"
scope = "revisions"
desc = "undo"
`)
	h.Start("initial")

	h.Key("Y")
	h.WaitText("Are you sure you want to undo")
	h.Text("?")
	if _, err := h.session.WaitForStableScreen(h.ctx, 3, func(screen []string) bool {
		return statusHasEntry(screen, "enter", "apply") &&
			!screenContains(screen, "Revset") &&
			!statusHasEntry(screen, "tab", "autocomplete")
	}); err != nil {
		screen, snapshotErr := h.session.Snapshot()
		if snapshotErr != nil {
			t.Fatalf("could not inspect expanded status help: %v", snapshotErr)
		}
		t.Fatalf("expanded undo help did not show undo actions or included dormant revset help:\n%s", formatScreen(screen))
	}

	h.Key("Escape")
	h.Key("Escape")
	h.WaitNoText("Are you sure you want to undo")
	h.Quit()
}
