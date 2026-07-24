//go:build e2e

package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	ghostty "go.mitchellh.com/libghostty"
)

func Test_Annotation_CreateEditDeleteAndCopy(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.repo.Write("review.go", "first line\nsecond line\nthird line\n").Commit("review target")
	openAnnotation(t, h, "review target", "review.go")
	h.Key("J") // Select the first two added lines.
	addReviewComment(t, h, "check both lines")
	screen := h.WaitText("comment: check both lines")
	assertCommentAfter(t, screen, "second line", "comment: check both lines")
	h.WaitText("1 annotations")

	h.Key("c")
	h.WaitText("alt+enter")
	h.Text(" updated")
	saveReviewComment(t, h)
	h.WaitText("comment: check both lines updated")
	h.WaitText("1 annotations")

	h.Key("y")
	payload := waitAnnotationClipboard(t, h)
	revision := strings.TrimSpace(runCommand(t, h.Repo(), h.Env(), "jj", "log", "-r", "@-", "--no-graph", "-T", "change_id.shortest()"))
	want := fmt.Sprintf("### @review.go (revision %s; new 1-2)\n\n```diff\n+first line\n+second line\n```\n\ncheck both lines updated", revision)
	if payload != want {
		t.Fatalf("clipboard = %q; want %q", payload, want)
	}
	h.Key("x")
	h.WaitText("0 annotations")
	h.WaitNoText("comment: check both lines")
	closeAnnotation(t, h)
}

func Test_Annotation_EditorConsumesBindingsAndCancelsDraft(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.repo.Write("review.go", "saved line\ndraft line\n").Commit("editor target")
	openAnnotation(t, h, "editor target", "review.go")
	addReviewComment(t, h, "saved comment")
	h.Key("j")
	h.Key("c")
	h.WaitText("alt+enter")
	h.Text("jkvwctxq{}[]")
	h.WaitText("jkvwctxq{}[]")
	h.Key("Escape")
	h.WaitText("comment: saved comment")
	h.WaitText("1 annotations")
	h.WaitNoText("jkvwctxq{}[]")
	// Cancelling an edit must preserve the previously saved value too.
	h.Key("k")
	h.Key("c")
	h.WaitText("alt+enter")
	h.Text(" discarded edit")
	h.WaitText("discarded edit")
	h.Key("Escape")
	h.WaitText("comment: saved comment")
	h.WaitNoText("discarded edit")
	closeAnnotation(t, h)
}

func Test_Annotation_CommentPickerFindsLineOutsideDiff(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	var content strings.Builder
	for line := 1; line <= 20; line++ {
		fmt.Fprintf(&content, "unchanged line %02d\n", line)
	}
	h.repo.Write("review.txt", content.String()).Commit("base content").
		Append("review.txt", "added at bottom\n").Commit("outside diff target")
	openAnnotation(t, h, "outside diff target", "review.txt")
	h.WaitNoText("unchanged line 01")
	h.Key("v")
	h.WaitText("review.txt [full]")
	h.WaitText("unchanged line 01")
	addReviewComment(t, h, "outside diff note")
	h.Key("v")
	h.WaitText("added at bottom")
	h.WaitNoText("comment: outside diff note")
	pickReviewComment(h, "outside diff note")
	screen := h.WaitText("comment: outside diff note")
	h.WaitText("review.txt [full]")
	assertCommentAfter(t, screen, "unchanged line 01", "comment: outside diff note")
	closeAnnotation(t, h)
}

func Test_Annotation_NavigatesFilesAndRevisionsWithComments(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.repo.Write("parent.txt", "parent source\n").Commit("parent review").
		Write("a.txt", "child source a\n").Write("b.txt", "child source b\n").Commit("child review")
	openAnnotation(t, h, "child review", "a.txt")
	addReviewComment(t, h, "child note a")
	annotationKey(t, h, ghostty.KeyBracketRight, "]", 0)
	h.WaitText("b.txt")
	h.WaitText("child source b")
	addReviewComment(t, h, "child note b")
	annotationKey(t, h, ghostty.KeyBracketLeft, "[", 0)
	h.WaitText("comment: child note a")
	// Exercise the real file picker and its return to the annotation scope.
	h.Text("\x14") // Ctrl+T opens the file picker.
	h.WaitText("target picker")
	h.WaitText("b.txt")
	h.Text("b.txt")
	h.Key("Enter")
	h.WaitText("comment: child note b")
	annotationKey(t, h, ghostty.KeyBracketRight, "}", ghostty.ModShift)
	h.WaitText("parent source")
	h.WaitText("parent review")
	addReviewComment(t, h, "parent note")
	annotationKey(t, h, ghostty.KeyBracketLeft, "{", ghostty.ModShift)
	h.WaitText("child source a")
	h.WaitText("2 annotations")
	pickReviewComment(h, "parent note")
	h.WaitText("comment: parent note")
	h.WaitText("parent.txt")
	h.WaitText("parent review")
	h.WaitText("1 annotations")
	pickReviewComment(h, "child note b")
	h.WaitText("comment: child note b")
	h.WaitText("b.txt")
	h.WaitText("child review")
	h.WaitText("2 annotations")
	h.Key("y")
	payload := waitAnnotationClipboard(t, h)
	for _, note := range []string{"child note a", "child note b", "parent note"} {
		if !strings.Contains(payload, note) {
			t.Fatalf("clipboard missing %q: %s", note, payload)
		}
	}
	h.Key("X")
	h.WaitText("0 annotations")
	h.Key("t")
	h.WaitText("No annotations")
	closeAnnotation(t, h)
}

func Test_Annotation_DeletedFileUsesOldLines(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	h.repo.Write("deleted.txt", "deleted first\ndeleted second\n").Commit("before deletion")
	if err := os.Remove(filepath.Join(h.Repo(), "deleted.txt")); err != nil {
		t.Fatal(err)
	}
	h.repo.Commit("deletion review")
	openAnnotation(t, h, "deletion review", "deleted.txt")
	h.Key("v")
	h.WaitText("deleted.txt [full]")
	h.WaitText("deleted first")
	addReviewComment(t, h, "old side note")
	h.Key("v")
	screen := h.WaitText("comment: old side note")
	assertCommentAfter(t, screen, "deleted first", "comment: old side note")
	h.Key("y")
	payload := waitAnnotationClipboard(t, h)
	if !strings.Contains(payload, "; old 1)") || !strings.Contains(payload, "\n deleted first\n") {
		t.Fatalf("deleted-file clipboard lost old-side location or snippet: %s", payload)
	}
	closeAnnotation(t, h)
}

func Test_Annotation_ResizePreservesEditorDraft(t *testing.T) {
	t.Parallel()
	h := NewHarness(t)
	var content strings.Builder
	for line := 1; line <= 45; line++ {
		fmt.Fprintf(&content, "source line %02d with enough text to wrap in a narrow terminal\n", line)
	}
	h.repo.Write("long.txt", content.String()).Commit("resize review")
	openAnnotation(t, h, "resize review", "long.txt")
	h.Key("w")
	h.Key("G")
	h.WaitText("source line 45")
	h.Key("c")
	h.WaitText("alt+enter")
	h.Text("draft survives resize and remains editable")
	h.WaitText("draft survives resize")
	if err := h.session.Resize(48, 12); err != nil {
		t.Fatal(err)
	}
	h.WaitStableText("draft survives resize", 3)
	cursor, err := h.session.Cursor()
	if err != nil {
		t.Fatal(err)
	}
	if !cursor.Visible || !cursor.InViewport {
		t.Fatalf("editor cursor not visible after resize: %+v", cursor)
	}
	h.Text(" done")
	saveReviewComment(t, h)
	h.WaitText("comment: draft survives")
	if err := h.session.Resize(100, 30); err != nil {
		t.Fatal(err)
	}
	screen := h.WaitText("comment: draft survives resize and remains editable done")
	assertCommentAfter(t, screen, "source line 45", "comment: draft survives resize")
	closeAnnotation(t, h)
}

func openAnnotation(t *testing.T, h *Harness, description, path string) {
	t.Helper()
	h.Start(description)
	h.Key("j") // The initial selection is the empty working-copy revision.
	h.Key("C")
	h.WaitText(path)
	h.WaitText("0 annotations")
}

func addReviewComment(t *testing.T, h *Harness, comment string) {
	t.Helper()
	h.Key("c")
	h.WaitText("alt+enter")
	h.Text(comment)
	saveReviewComment(t, h)
	h.WaitText("comment: " + comment)
}

func saveReviewComment(t *testing.T, h *Harness) {
	t.Helper()
	annotationKey(t, h, ghostty.KeyEnter, "", ghostty.ModAlt)
	h.WaitNoText("alt+enter")
}

func annotationKey(t *testing.T, h *Harness, key ghostty.Key, text string, mods ghostty.Mods) {
	t.Helper()
	if err := h.session.SendKey(key, text, mods); err != nil {
		t.Fatal(err)
	}
}

func pickReviewComment(h *Harness, comment string) {
	h.Key("t")
	h.WaitText("target picker")
	h.WaitText(comment)
	h.Text(comment)
	h.Key("Enter")
}

func closeAnnotation(t *testing.T, h *Harness) {
	t.Helper()
	h.Key("Escape")
	h.WaitNoText("annotations")
	h.Quit()
}

func assertCommentAfter(t *testing.T, screen []string, source, comment string) {
	t.Helper()
	sourceRow := screenRowIndexContaining(screen, source)
	commentRow := screenRowIndexContaining(screen, comment)
	if sourceRow < 0 || commentRow != sourceRow+1 {
		t.Fatalf("comment %q must appear immediately after %q:\n%s", comment, source, formatScreen(screen))
	}
}

func waitAnnotationClipboard(t *testing.T, h *Harness) string {
	t.Helper()
	// Inspect OSC 52 output without touching the host clipboard.
	pattern := regexp.MustCompile("\x1b\\]52;c;([A-Za-z0-9+/=]*)(?:\x07|\x1b\\\\)")
	var payload string
	waitFor(t, h.ctx, func() (bool, error) {
		matches := pattern.FindAllStringSubmatch(h.session.RawOutput(), -1)
		if len(matches) == 0 {
			return false, nil
		}
		decoded, err := base64.StdEncoding.DecodeString(matches[len(matches)-1][1])
		payload = string(decoded)
		return true, err
	})
	return payload
}
