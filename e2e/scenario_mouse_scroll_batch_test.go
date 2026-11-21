//go:build e2e

package main

import (
	"fmt"
	"testing"
)

// Loading the next log batch while wheel-scrolling must not snap the view back to the cursor (#785).
func Test_Revisions_WheelScrollKeepsPositionWhenBatchLoads(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	for i := 1; i <= 40; i++ {
		h.repo.JJ("new", "-m", fmt.Sprintf("row-%02d", i))
	}
	h.WriteConfigTOML("[revisions]\nlog_batch_size = 10\n")

	h.Start("row-40")

	const wheelDown = "\x1b[<65;10;10M"
	scrolledAway := false
	for tick := 0; tick < 30; tick++ {
		h.Text(wheelDown)
		screen, err := h.session.WaitForStableScreen(h.ctx, 10, func([]string) bool { return true })
		if err != nil {
			h.t.Fatal(err)
		}
		if screenContains(screen, "row-40") {
			if scrolledAway {
				h.t.Fatalf("view jumped back to the cursor after tick %d:\n%s", tick, formatScreen(screen))
			}
			continue
		}
		scrolledAway = true
		if screenContains(screen, "row-05") {
			break
		}
	}
	if !scrolledAway {
		h.t.Fatal("wheel scrolling never moved the view")
	}
	h.WaitText("row-05")

	h.Quit()
}
