//go:build e2e

package main

import (
	"strings"
	"testing"
)

func Test_Revisions_NewRevisionKeepsSelectionOrderForParents(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.repo.
		Append("README.md", "older\n").
		Commit("older").
		New("@--").
		Write("other.txt", "newer\n").
		Commit("newer")

	h.Start("older")
	h.WaitTextInOrder("newer", "older")

	// Check older first, then move up and check newer. The selection order
	// is the reverse of the log order. The default space action also jumps
	// to the parent, so after checking older the cursor is on initial.
	h.Key("j")
	h.Key("j")
	h.Key(" ")
	waitForCheckedRevisions(h, "older")
	h.Key("k")
	h.Key("k")
	h.Key(" ")
	waitForCheckedRevisions(h, "newer", "older")
	h.Key("n")

	var parents string
	waitFor(t, h.ctx, func() (bool, error) {
		out, err := commandOutput(h.Repo(), h.Env(), "jj", "log", "-r", "@", "--no-graph", "--color", "never",
			"-T", `parents.map(|c| c.description().first_line()).join(",")`)
		if err != nil {
			return false, nil
		}
		parents = strings.TrimSpace(out)
		return strings.Contains(parents, ","), nil
	})

	if parents != "older,newer" {
		t.Fatalf("parents of new revision = %q, want %q (selection order)", parents, "older,newer")
	}

	h.Quit()
}

// The checkmark is on the header row, directly above the description.
func waitForCheckedRevisions(h *Harness, descriptions ...string) {
	h.t.Helper()
	_, err := h.session.WaitForStableScreen(h.ctx, 3, func(screen []string) bool {
		checked := 0
		for _, row := range screen {
			if strings.Contains(row, "✓") {
				checked++
			}
		}
		if checked != len(descriptions) {
			return false
		}
		for _, description := range descriptions {
			row := screenRowIndexContaining(screen, description)
			if row < 1 || !strings.Contains(screen[row-1], "✓") {
				return false
			}
		}
		return true
	})
	if err != nil {
		h.t.Fatalf("waiting for checked revisions %q: %v", descriptions, err)
	}
}
