//go:build e2e

package main

import (
	"fmt"
	"strings"
	"testing"
)

func Test_FileView_AnnotatesAndStepsBackThroughHistory(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	annotateWithDescriptions(h)
	h.repo.Write("a.txt", "one\n").Commit("add one")
	h.repo.Write("a.txt", "one\ntwo\n").Commit("add two")
	h.Start("add two")

	h.Key("ctrl+t")
	h.WaitText("file:")
	h.Text("a.txt")
	h.WaitTextInOrder("file: a.txt", "a.txt")
	h.Key("Enter")
	screen := h.WaitTextInOrder("file: a.txt", "add one", "one", "add two", "two")
	t.Log("\n" + strings.Join(screen, "\n"))

	// the cursor is on "one"; step to the change that added "two" and go back
	// to before it
	h.Text("J")
	h.Text("{")
	h.WaitText("(1 back)")
	h.WaitNoText("add two")

	h.Text("}")
	h.WaitNoText("(1 back)")
	h.WaitText("add two")

	h.Text("j")
	h.Text("p")
	h.WaitText("regular file a.txt")

	h.Text("k")
	h.Key("Enter")
	h.WaitNoText("file: a.txt")
	h.WaitStyled(func(screen styledScreen) bool {
		return rowHasBackground(screen, "add one") && !rowHasBackground(screen, "add two")
	})
	h.Quit()
}

// annotateWithDescriptions makes the repo's annotation template print the
// descriptions the tests look for; jj's default only prints change ids.
func annotateWithDescriptions(h *Harness) {
	h.repo.JJ("config", "set", "--repo", "templates.file_annotate",
		`join(" ", commit.change_id().shortest(8), commit.description().first_line(), pad_start(4, line_number)) ++ ": " ++ content`)
}

func rowHasBackground(screen styledScreen, text string) bool {
	for _, row := range screen {
		var line strings.Builder
		background := false
		for _, cell := range row {
			line.WriteString(cell.Text)
			background = background || cell.Style.HasBackground
		}
		if strings.Contains(line.String(), text) {
			return background
		}
	}
	return false
}

func Test_FileView_OpensTheFileFocusedInDetails(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	annotateWithDescriptions(h)
	h.repo.Write("b.txt", "beta\n").Commit("add b")
	h.Start("add b")

	h.Key("j")
	h.Text("l")
	h.WaitText("change id")
	h.Text("j")
	h.Key("ctrl+t")
	h.WaitTextInOrder("file: b.txt", "add b", "beta")
	h.Quit()
}

func Test_FileView_JumpAddsAHiddenChangeToTheRevset(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	annotateWithDescriptions(h)
	h.WriteConfigTOML("[revisions]\nrevset = \"@ | @-\"\n")
	h.repo.Write("a.txt", "one\n").Commit("add one")
	h.repo.Write("a.txt", "one\ntwo\n").Commit("add two")
	h.Start("add two")
	h.WaitNoText("add one")

	h.Key("ctrl+t")
	h.Text("a.txt")
	h.WaitTextInOrder("file: a.txt", "a.txt")
	h.Key("Enter")
	h.WaitTextInOrder("file: a.txt", "add one", "one")

	// the cursor is on "one", added by the hidden change
	h.Key("Enter")
	h.WaitText("revset: (@ | @-) | ")
	h.WaitStyled(func(screen styledScreen) bool {
		return rowHasBackground(screen, "add one") && !rowHasBackground(screen, "add two")
	})
	h.Quit()
}

func Test_FileView_JumpSelectsAChangeBeyondTheFirstBatch(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	annotateWithDescriptions(h)
	h.repo.Write("a.txt", "old\n").Commit("add old")
	for i := 1; i <= 40; i++ {
		h.repo.JJ("new", "-m", fmt.Sprintf("row-%02d", i))
	}
	h.WriteConfigTOML("[revisions]\nlog_batch_size = 10\n")
	h.Start("row-40")

	h.Key("ctrl+t")
	h.WaitText("file:")
	h.Text("a.txt")
	h.WaitTextInOrder("file: a.txt", "a.txt")
	h.Key("Enter")
	h.WaitTextInOrder("file: a.txt", "add old", "old")
	h.Key("Enter")
	h.WaitNoText("file: a.txt")
	h.WaitStyled(func(screen styledScreen) bool {
		return rowHasBackground(screen, "add old")
	})
	h.Quit()
}

func Test_FileView_SearchesTheFileText(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.repo.Write("a.txt", "apple\nbanana\ngrape\nbanana split\n").Commit("add fruit")
	h.Start("add fruit")

	h.Key("ctrl+t")
	h.WaitText("file:")
	h.Text("a.txt")
	h.WaitTextInOrder("file: a.txt", "a.txt")
	h.Key("Enter")
	h.WaitText("banana split")

	h.Text("/")
	h.WaitText("search:")
	h.Text("banana")
	h.Key("Enter")
	h.WaitText("search: banana")
	h.WaitStyled(func(screen styledScreen) bool {
		return rowHasBackground(screen, "2: banana") && inverted(screen, "banana") == 2
	})

	h.Text("'")
	h.WaitStyled(func(screen styledScreen) bool {
		return rowHasBackground(screen, "banana split") && !rowHasBackground(screen, "2: banana")
	})

	h.Key("Escape")
	h.WaitNoText("search: banana")
	h.WaitText("file: a.txt")
	h.Key("Escape")
	h.WaitNoText("file: a.txt")
	h.Quit()
}

// inverted counts the runs of inverse cells that spell text.
func inverted(screen styledScreen, text string) int {
	count := 0
	for _, row := range screen {
		var run strings.Builder
		for _, cell := range append(row, screenCell{}) {
			if cell.Style.Inverse {
				run.WriteString(cell.Text)
				continue
			}
			if run.String() == text {
				count++
			}
			run.Reset()
		}
	}
	return count
}
