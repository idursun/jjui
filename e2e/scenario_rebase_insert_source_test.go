//go:build e2e

package main

import (
	"strings"
	"testing"
)

func Test_Revisions_RebasesSourceWithInsertBetween(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.repo.
		Append("README.md", "base\n").
		Commit("base").
		Append("README.md", "source\n").
		Commit("source").
		New("@--").
		Append("README.md", "target\n").
		Commit("target")

	h.Start("target")
	h.WaitTextInOrder("target", "source", "base")

	// Space runs asynchronously, so wait for each state before the next key.
	h.Key("j")
	h.Key("j")
	h.Key(" ")
	waitForCheckedCount(h, 1)

	// Space moved the cursor down to base, which becomes the insert-after end.
	h.Key("r")
	h.WaitText("<< onto >>")
	h.Key("s")
	h.Key("i")
	h.WaitText("between")
	waitForMarkerAbove(h, "<< before this >>", "base")

	h.Key("k")
	waitForMarkerAbove(h, "<< before this >>", "source")
	h.Key("k")
	waitForMarkerAbove(h, "<< before this >>", "target")
	waitForMarkerAbove(h, "<< after this >>", "base")
	h.Key("Enter")

	waitFor(t, h.ctx, func() (bool, error) {
		return parentDescription(h, "target") == "source" && parentDescription(h, "source") == "base", nil
	})

	h.Quit()
}

// The marker is on the header row, directly above the description.
func waitForMarkerAbove(h *Harness, marker, description string) {
	h.t.Helper()
	_, err := h.session.WaitForScreen(h.ctx, func(screen []string) bool {
		row := screenRowIndexContaining(screen, description)
		return row >= 1 && strings.Contains(screen[row-1], marker)
	})
	if err != nil {
		h.t.Fatalf("waiting for %q above %q: %v", marker, description, err)
	}
}

func parentDescription(h *Harness, description string) string {
	out, err := commandOutput(h.Repo(), h.Env(), "jj", "log", "--no-graph", "--color", "never",
		"-r", `description(exact:"`+description+`\n")`,
		"-T", `parents.map(|c| c.description().first_line()).join(",")`)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}
