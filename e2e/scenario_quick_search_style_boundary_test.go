//go:build e2e

package main

import (
	"strings"
	"testing"
)

// Issue #759: IDs render as a highlighted unique prefix plus a dimmed rest, so a
// query spanning both parts must still match.
func Test_QuickSearch_MatchesAcrossChangeIdStyleBoundary(t *testing.T) {
	t.Parallel()
	assertQuickSearchCrossesStyleBoundary(t, "change_id.shortest(8)")
}

func Test_QuickSearch_MatchesAcrossCommitIdStyleBoundary(t *testing.T) {
	t.Parallel()
	assertQuickSearchCrossesStyleBoundary(t, "commit_id.shortest(8)")
}

func assertQuickSearchCrossesStyleBoundary(t *testing.T, idTemplate string) {
	t.Helper()
	h := NewHarness(t)
	h.repo.
		Append("README.md", "target\n").
		Commit("target").
		Append("README.md", "newer\n").
		Commit("newer")

	id := strings.TrimSpace(runCommand(t, h.Repo(), h.Env(), "jj", "log", "--no-graph", "-r", "description(exact:'target\n')", "-T", idTemplate))
	if len(id) != 8 {
		t.Fatalf("unexpected id %q", id)
	}

	h.Start("target")
	h.WaitTextInOrder("newer", "target", "initial")

	h.Key("/")
	h.Text(id)
	h.Key("Enter")
	h.WaitText("quick search")
	h.WaitStyled(func(screen styledScreen) bool {
		return screenHasInverseText(screen, id)
	})

	h.Key(" ")
	waitForCheckedCount(h, 1)

	h.Quit()
}

// screenHasInverseText reports whether want appears in consecutive single-character cells that are all inverse.
func screenHasInverseText(screen styledScreen, want string) bool {
	for _, row := range screen {
		for start := 0; start+len(want) <= len(row); start++ {
			matched := true
			for i := range len(want) {
				cell := row[start+i]
				if cell.Text != want[i:i+1] || !cell.Style.Inverse {
					matched = false
					break
				}
			}
			if matched {
				return true
			}
		}
	}
	return false
}
