//go:build e2e

package main

import (
	"fmt"
	"strings"
	"testing"
)

// The preview pane cannot take focus, so its hotkeys act on it while the
// primary view is active. They must not reach it from views drawn over it.

func startPreviewHotkeysRepo(t *testing.T) *Harness {
	t.Helper()
	h := NewHarness(t)
	var content strings.Builder
	for i := 1; i <= 120; i++ {
		fmt.Fprintf(&content, "preview-line-%03d\n", i)
	}
	h.repo.Write("long.txt", content.String()).Commit("preview hotkeys target")
	h.Start("preview hotkeys target")
	h.Key("j")
	h.Key("p")
	h.WaitText("preview-line-001")
	return h
}

func (h *Harness) KeyTimes(name string, times int) {
	h.t.Helper()
	for range times {
		h.Key(name)
	}
}

func Test_PreviewHotkeys_ScrollPreviewFromRevisions(t *testing.T) {
	t.Parallel()
	h := startPreviewHotkeysRepo(t)

	h.KeyTimes("ctrl+n", 20)
	h.WaitNoText("preview-line-001")
	h.Quit()
}

func Test_PreviewHotkeys_ScrollPreviewFromDetails(t *testing.T) {
	t.Parallel()
	h := startPreviewHotkeysRepo(t)

	h.Key("l")
	h.WaitMode("details")
	// Wait for the file preview to replace the revision preview.
	h.WaitStableText("Added regular file long.txt:", 3)
	h.KeyTimes("ctrl+n", 20)
	h.WaitNoText("preview-line-001")
	h.Quit()
}

func Test_PreviewHotkeys_DoNotReachPreviewUnderDiff(t *testing.T) {
	t.Parallel()
	h := startPreviewHotkeysRepo(t)

	h.Key("d")
	h.WaitMode("diff")
	// The diff does not bind ctrl+n; it must not scroll the hidden preview.
	h.KeyTimes("ctrl+n", 20)
	h.WaitStableMode("diff")
	h.Key("Escape")
	h.WaitMode("revisions")
	h.WaitStableText("preview-line-001", 3)
	h.Quit()
}
