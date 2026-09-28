//go:build e2e

package main

import (
	"strings"
	"testing"
)

// These scenarios pin how the root UI stacks its views (revisions, oplog,
// diff and dialogs): which one is shown, which one receives keys, and which
// one closes first.

func statusMode(screen []string) string {
	for i := len(screen) - 1; i >= 0; i-- {
		if fields := strings.Fields(screen[i]); len(fields) > 0 {
			return fields[0]
		}
	}
	return ""
}

func (h *Harness) WaitMode(mode string) []string {
	h.t.Helper()
	screen, err := h.session.WaitForScreen(h.ctx, func(screen []string) bool {
		return statusMode(screen) == mode
	})
	if err != nil {
		h.t.Fatalf("waiting for mode %q: %v", mode, err)
	}
	return screen
}

func (h *Harness) WaitStableMode(mode string) []string {
	h.t.Helper()
	screen, err := h.session.WaitForStableScreen(h.ctx, 3, func(screen []string) bool {
		return statusMode(screen) == mode
	})
	if err != nil {
		h.t.Fatalf("waiting for stable mode %q: %v", mode, err)
	}
	return screen
}

func startViewStackRepo(t *testing.T) *Harness {
	t.Helper()
	h := NewHarness(t)
	h.repo.Write("a.txt", "view stack content\n").Commit("view stack target")
	return h
}

func openTargetDiff(h *Harness) {
	h.t.Helper()
	h.Key("j")
	h.Key("d")
	h.WaitMode("diff")
	h.WaitText("view stack content")
}

func Test_ViewStack_DiffClosesBackToRevisions(t *testing.T) {
	t.Parallel()
	h := startViewStackRepo(t)
	h.Start("view stack target")

	openTargetDiff(h)
	h.Key("Escape")
	h.WaitMode("revisions")
	h.WaitText("view stack target")
	h.Quit()
}

func Test_ViewStack_OpLogDiffClosesBackToOpLog(t *testing.T) {
	t.Parallel()
	h := startViewStackRepo(t)
	h.Start("view stack target")

	h.Key("o")
	h.WaitMode("oplog")
	h.WaitText("args: jj commit -m 'view stack target'")
	h.Key("d")
	h.WaitMode("diff")

	h.Key("Escape")
	h.WaitMode("oplog")
	h.Key("Escape")
	h.WaitMode("revisions")
	h.Quit()
}

func Test_ViewStack_HelpDoesNotOpenOverDiff(t *testing.T) {
	t.Parallel()
	h := startViewStackRepo(t)
	h.Start("view stack target")

	openTargetDiff(h)
	h.Key("F1")
	h.WaitStableMode("diff")

	h.Key("Escape")
	h.WaitMode("revisions")
	h.Quit()
}

func Test_ViewStack_DialogKeysDoNotReachRevisions(t *testing.T) {
	t.Parallel()
	h := startViewStackRepo(t)
	h.Start("view stack target")
	before := countRevisions(t, h.Repo(), h.Env())

	h.Key("b")
	h.WaitText("Bookmark Operations")
	// "n" is revisions.new; the bookmarks dialog leaves it unbound.
	h.Key("n")
	h.WaitStableText("Bookmark Operations", 3)
	h.Key("Escape")
	h.WaitNoText("Bookmark Operations")
	h.WaitMode("revisions")

	if after := countRevisions(t, h.Repo(), h.Env()); after != before {
		t.Fatalf("key under the bookmarks dialog reached revisions: %d revisions, want %d", after, before)
	}
	h.Quit()
}

func Test_ViewStack_CommandHistoryHidesFlashMessages(t *testing.T) {
	t.Parallel()
	h := startViewStackRepo(t)
	h.WriteConfigLua(`
function setup(config)
  config.action("view-stack-flash", function()
    flash({text = "view-stack-flash-message", error = true})
  end, { key = "Y", scope = "revisions" })
end
`)
	h.Start("view stack target")

	h.Key("Y")
	h.WaitText("view-stack-flash-message")
	h.Key("W")
	h.WaitMode("command")
	h.WaitNoText("view-stack-flash-message")

	h.Key("Escape")
	h.WaitMode("revisions")
	h.WaitText("view-stack-flash-message")
	h.Quit()
}

// Dialogs can open over a diff through ui-scope bindings. The dialog is drawn
// on top of the diff, so it should also be the first to close.
func Test_ViewStack_DialogOverDiffClosesFirst(t *testing.T) {
	t.Parallel()
	h := startViewStackRepo(t)
	h.WriteConfigTOML(`
[[bindings]]
action = "ui.open_git"
key = "Y"
scope = "ui"
desc = "git"
`)
	h.Start("view stack target")

	openTargetDiff(h)
	h.Key("Y")
	h.WaitText("Git Operations")

	h.Key("Escape")
	h.WaitNoText("Git Operations")
	h.WaitMode("diff")
	h.WaitText("view stack content")

	h.Key("Escape")
	h.WaitMode("revisions")
	h.Quit()
}

// The oplog can be opened over a diff through ui-scope bindings.
func Test_ViewStack_OpLogOpenedOverDiff(t *testing.T) {
	t.Parallel()
	h := startViewStackRepo(t)
	h.WriteConfigTOML(`
[[bindings]]
action = "ui.open_oplog"
key = "Y"
scope = "ui"
desc = "oplog"
`)
	h.Start("view stack target")

	openTargetDiff(h)
	h.Key("Y")
	// Current behaviour: the diff stays in front of the oplog opened after it.
	h.WaitStableMode("diff")

	h.Key("Escape")
	h.WaitMode("oplog")
	h.Key("Escape")
	h.WaitMode("revisions")
	h.Quit()
}
