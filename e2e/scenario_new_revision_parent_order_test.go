//go:build e2e

package main

import (
	"fmt"
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

	// Check older, then newer: the reverse of log order. Space moves down, so it takes two k to reach newer.
	h.Key("j")
	h.Key("j")
	h.Key(" ")
	waitForCheckedCount(h, 1)
	h.Key("k")
	h.Key("k")
	h.Key(" ")
	waitForCheckedCount(h, 2)
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

func waitForCheckedCount(h *Harness, count int) {
	h.t.Helper()
	want := fmt.Sprintf("%d checked", count)
	_, err := h.session.WaitForStableScreen(h.ctx, 3, func(screen []string) bool {
		return screenContains(screen, want)
	})
	if err != nil {
		h.t.Fatalf("waiting for %q: %v", want, err)
	}
}
