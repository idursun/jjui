package annotation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/jj/source"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations/target_picker"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelLoadsStructuredDiffAndCompleteFile(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	defer runner.Verify()
	commit := jj.Commit{ChangeId: "change", CommitId: "commit"}
	runner.Expect(jj.GetDescription("change")).SetOutput([]byte("Replace old with new"))
	runner.Expect(jj.AnnotationDiff("change")).SetOutput([]byte(`diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1 +1 @@
-old
+new`))
	runner.Expect(jj.FileShow("change", "a.go")).SetOutput([]byte("new\nunchanged line"))

	model := New(test.NewTestContext(runner), commit.ChangeId)
	cmd := model.Init()
	require.NotNil(t, cmd)
	require.Nil(t, model.Update(cmd()))

	diffView := test.Stripped(test.RenderImmediate(model, 80, 10))
	assert.NotContains(t, diffView, "ANNOTATE")
	assert.Contains(t, diffView, "change · Replace old with new")
	assert.NotContains(t, diffView, "diff against parent")
	assert.Contains(t, diffView, "a.go")
	assert.Contains(t, diffView, "old")
	assert.Contains(t, diffView, "new")
	assert.Contains(t, diffView, "1")

	cmd, handled := model.HandleIntent(intents.AnnotationTogglePresentation{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	require.Nil(t, model.Update(cmd()))

	fileView := test.Stripped(ansi.Strip(test.RenderImmediate(model, 80, 10)))
	assert.Contains(t, fileView, "a.go [full]")
	assert.Contains(t, fileView, "unchanged line")
}

func TestAnnotationsRenderOnUnchangedLines(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	defer runner.Verify()
	commit := jj.Commit{ChangeId: "change", CommitId: "commit"}
	expectRevisionLoad := func() {
		runner.Expect(jj.GetDescription("change")).SetOutput([]byte("Keep unchanged lines visible"))
		runner.Expect(jj.AnnotationDiff("change")).SetOutput([]byte(`diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1 +1 @@
 unchanged`))
	}
	expectRevisionLoad()
	runner.Expect(jj.FileShow("change", "a.go")).SetOutput([]byte("unchanged\noutside hunk"))

	model := New(test.NewTestContext(runner), commit.ChangeId)
	require.Nil(t, model.Update(model.Init()()))
	cmd, _ := model.HandleIntent(intents.AnnotationTogglePresentation{})
	require.Nil(t, model.Update(cmd()))
	model.startEditor()
	model.editor.SetValue("review this unchanged line")
	require.Nil(t, model.saveEditor())
	assert.Len(t, model.annotations.All(), 1)
	rendered := test.Stripped(ansi.Strip(test.RenderImmediate(model, 70, 12)))
	assert.Contains(t, rendered, "comment: review this unchanged line")
}

func TestNavigateRevisionMovesToSingleDirectChildAndKeepsModelAnnotations(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	defer runner.Verify()
	first := jj.Commit{ChangeId: "first", CommitId: "111111111111"}
	second := jj.Commit{ChangeId: "second", CommitId: "222222222222"}
	runner.Expect(jj.GetDescription(first.ChangeId)).SetOutput([]byte("First revision"))
	runner.Expect(jj.GetDescription(second.ChangeId)).SetOutput([]byte("Second revision"))
	runner.Expect(jj.AnnotationDiff(first.ChangeId)).SetOutput([]byte("diff --git a/a.go b/a.go\n"))
	runner.Expect(jj.AnnotationDiff(second.ChangeId)).SetOutput([]byte("diff --git a/b.go b/b.go\n"))
	runner.Expect(jj.GetRevisionSummariesFromRevset(first.ChangeId + "+")).SetOutput([]byte(
		"second\tSecond revision\n",
	))

	model := New(test.NewTestContext(runner), first.ChangeId)
	model.addAnnotation(Annotation{ChangeID: first.ChangeId, File: "a.go", Comment: "first note"})
	require.Nil(t, model.Update(model.Init()()))
	beforeLoad := test.RenderImmediate(model, 80, 10)

	cmd, handled := model.HandleIntent(intents.AnnotationNavigateChild{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	load := model.Update(cmd())
	require.NotNil(t, load)
	assert.Equal(t, first.ChangeId, model.changeID())
	assert.Equal(t, "a.go", model.currentFile().Path)
	assert.Equal(t, beforeLoad, test.RenderImmediate(model, 80, 10))
	require.Nil(t, model.Update(load()))

	assert.Equal(t, second.ChangeId, model.changeID())
	assert.Len(t, model.annotations.All(), 1)
	assert.Equal(t, "b.go", model.currentFile().Path)
}

func TestNavigateRevisionOpensPickerForMultipleDirectParents(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	defer runner.Verify()
	current := jj.Commit{ChangeId: "current", CommitId: "333333333333"}
	first := jj.Commit{ChangeId: "first-parent", CommitId: "111111111111"}
	second := jj.Commit{ChangeId: "second-parent", CommitId: "222222222222"}
	runner.Expect(jj.GetRevisionSummariesFromRevset(current.ChangeId + "-")).SetOutput([]byte(
		"first-parent\tMainline work\nsecond-parent\tFeature work\n",
	))

	model := New(test.NewTestContext(runner), current.ChangeId)
	cmd, handled := model.HandleIntent(intents.AnnotationNavigateParent{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	openPicker := model.Update(cmd())
	require.NotNil(t, openPicker)
	message, ok := openPicker().(common.OpenTargetPickerMsg)
	require.True(t, ok)
	require.Len(t, message.Sources, 1)
	items, err := message.Sources[0].Fetch(nil)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "first-parent · Mainline work", items[0].Name)
	assert.Equal(t, "first-parent", items[0].Value)
	assert.Equal(t, source.KindRevision, items[0].Kind)
	assert.Equal(t, "second-parent · Feature work", items[1].Name)
	assert.Equal(t, "second-parent", items[1].Value)
	assert.Equal(t, source.KindRevision, items[1].Kind)

	load := model.Update(target_picker.TargetSelectedMsg{
		Target:  items[1].Value,
		Payload: message.Payload,
	})
	require.NotNil(t, load)
	assert.Equal(t, current.ChangeId, model.document.revision)
	assert.Equal(t, second.ChangeId, model.document.loadingRevision)
	assert.NotEqual(t, first.ChangeId, model.document.loadingRevision)
}

func TestNavigateRevisionStaysPutWhenThereAreNoDirectTargets(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	defer runner.Verify()
	current := jj.Commit{ChangeId: "current", CommitId: "333333333333"}
	runner.Expect(jj.GetRevisionSummariesFromRevset(current.ChangeId + "-")).SetOutput(nil)

	model := New(test.NewTestContext(runner), current.ChangeId)
	cmd, handled := model.HandleIntent(intents.AnnotationNavigateParent{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	assert.Nil(t, model.Update(cmd()))
	assert.Equal(t, current.ChangeId, model.document.revision)
}

func TestFailedRevisionLoadKeepsCurrentRevisionVisible(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	defer runner.Verify()
	runner.Expect(jj.GetDescription("current")).SetOutput([]byte("Current revision"))
	runner.Expect(jj.AnnotationDiff("current")).SetOutput([]byte("diff --git a/a.go b/a.go\n"))
	runner.Expect(jj.GetDescription("target")).SetOutput([]byte("Target revision"))
	runner.Expect(jj.AnnotationDiff("target")).SetError(errors.New("target load failed"))

	model := New(test.NewTestContext(runner), "current")
	require.Nil(t, model.Update(model.Init()()))
	beforeLoad := test.RenderImmediate(model, 80, 10)

	load := model.selectRevision("target")
	require.NotNil(t, load)
	showError := model.Update(load())
	require.NotNil(t, showError)

	assert.Equal(t, "current", model.changeID())
	assert.Empty(t, model.document.loadingRevision)
	assert.Equal(t, beforeLoad, test.RenderImmediate(model, 80, 10))
	message, ok := showError().(intents.AddMessage)
	require.True(t, ok)
	assert.Equal(t, "target load failed", message.Text)
}

func TestHeaderShowsFileCountsAnnotationsAndRevisionOnOneRow(t *testing.T) {
	current := jj.Commit{ChangeId: "current", CommitId: "222222222222"}
	model := New(nil, current.ChangeId)
	model.document.description = "Current revision description"
	model.document.files = []fileItem{{Path: "a.go"}, {Path: "b.go"}}
	model.document.file = 1
	model.addAnnotation(Annotation{ChangeID: current.ChangeId})
	model.addAnnotation(Annotation{ChangeID: current.ChangeId})
	rendered := ansi.Strip(test.RenderImmediate(model, 80, 2))
	header := strings.TrimSpace(strings.SplitN(rendered, "\n", 2)[0])
	assert.Equal(
		t,
		"b.go · file 2/2 · 2 annotations │ current · Current revision description",
		header,
	)
}

func TestPureRenameRendersJjStylePath(t *testing.T) {
	model := New(nil, "")
	model.document.revision = "change"
	model.document.files = patchFiles(parseGitPatch(`diff --git a/old.go b/new.go
rename from old.go
rename to new.go`))

	rendered := ansi.Strip(test.RenderImmediate(model, 80, 4))

	assert.Contains(t, rendered, "old.go -> new.go")
	assert.NotContains(t, rendered, "rename from")
	assert.NotContains(t, rendered, "rename to")
	assert.NotContains(t, rendered, "(unchanged file")
}

func TestTargetPickerListsChangedFilesAndSelectsOne(t *testing.T) {
	model := &Model{
		document: reviewDocument{
			files: []fileItem{
				{Path: "first.go"},
				{Path: "second.go"},
			},
		},
	}

	cmd, handled := model.HandleIntent(intents.AnnotationOpenTargetPicker{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	message, ok := cmd().(common.OpenTargetPickerMsg)
	require.True(t, ok)
	require.Len(t, message.Sources, 1)
	items, err := message.Sources[0].Fetch(nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"first.go", "second.go"}, []string{items[0].Name, items[1].Name})

	model.Update(target_picker.TargetSelectedMsg{Target: "second.go", Payload: targetPickerPayload{}})
	assert.Equal(t, 1, model.document.file)
}

func TestCommentPickerListsModelAnnotationsWithDistinctTargets(t *testing.T) {
	first := jj.Commit{ChangeId: "first-change", CommitId: "first"}
	second := jj.Commit{ChangeId: "second-change", CommitId: "second"}
	model := New(nil, first.ChangeId)
	model.addAnnotation(Annotation{ChangeID: first.ChangeId, File: "a.go", NewLines: lineRange{Start: 3, End: 3}, Comment: "same note"})
	model.addAnnotation(Annotation{ChangeID: first.ChangeId, File: "a.go", NewLines: lineRange{Start: 3, End: 3}, Comment: "same note"})
	model.addAnnotation(Annotation{ChangeID: second.ChangeId, File: "a.go", NewLines: lineRange{Start: 3, End: 3}, Comment: "same note"})

	cmd, handled := model.HandleIntent(intents.AnnotationOpenCommentPicker{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	message, ok := cmd().(common.OpenTargetPickerMsg)
	require.True(t, ok)
	require.Len(t, message.Sources, 1)
	items, err := message.Sources[0].Fetch(nil)
	require.NoError(t, err)
	require.Len(t, items, 3)
	assert.Equal(t, source.KindComment, items[0].Kind)
	assert.Contains(t, items[0].Name, "first-change · a.go:3 · same note")
	assert.Contains(t, items[1].Name, "first-change · a.go:3 · same note [2]")
	assert.Contains(t, items[2].Name, "second-chang · a.go:3 · same note")
	assert.Equal(t, "1", items[0].Value)
	assert.Equal(t, "2", items[1].Value)
	assert.Equal(t, "3", items[2].Value)
	assert.NotEqual(t, items[0].Name, items[1].Name)
}

func TestCommentPickerSelectionNavigatesToAnnotationAnchor(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	defer runner.Verify()
	first := jj.Commit{ChangeId: "first", CommitId: "first"}
	second := jj.Commit{ChangeId: "second", CommitId: "second"}
	runner.Expect(jj.GetDescription(second.ChangeId)).SetOutput([]byte("Second revision"))
	runner.Expect(jj.AnnotationDiff(second.ChangeId)).SetOutput([]byte(`diff --git a/b.go b/b.go
--- a/b.go
+++ b/b.go
@@ -1,2 +1,2 @@
 one
 two`))
	model := New(test.NewTestContext(runner), first.ChangeId)
	annotation := model.addAnnotation(Annotation{
		ChangeID: second.ChangeId,
		File:     "b.go",
		NewLines: lineRange{Start: 2, End: 2},
		Comment:  "review second line",
	})
	cmd := model.Update(target_picker.TargetSelectedMsg{
		Target:  strconv.Itoa(annotation.ID),
		Payload: commentPickerPayload{},
	})
	require.NotNil(t, cmd)
	require.Nil(t, model.Update(cmd()))

	assert.Equal(t, second.ChangeId, model.changeID())
	assert.Equal(t, diffPresentation, model.document.presentation)
	assert.Equal(t, "b.go", model.currentFile().Path)
	assert.Equal(t, 2, model.cursor)
	assert.Equal(t, annotation.ID, model.focusedAnnotationID)
}

func TestFocusedAnnotationCanBeEditedOrDeleted(t *testing.T) {
	model := annotationTestModel()
	annotation := model.addAnnotation(Annotation{ChangeID: "commit", File: "a.go", NewLines: lineRange{Start: 1, End: 1}, Comment: "before"})

	_, handled := model.HandleIntent(intents.AnnotationAdd{})
	assert.True(t, handled)
	assert.True(t, model.editing)
	assert.Equal(t, annotation.ID, model.focusedAnnotationID)
	assert.Equal(t, "before", model.editor.Value())
	model.editor.SetValue("after")
	require.Nil(t, model.saveEditor())
	updated, ok := model.annotationByID(annotation.ID)
	require.True(t, ok)
	assert.Equal(t, "after", updated.Comment)
	assert.Len(t, model.annotations.All(), 1)

	_, handled = model.HandleIntent(intents.AnnotationDelete{})
	assert.True(t, handled)
	assert.Empty(t, model.annotations.All())
}

func TestFocusedAnnotationDoesNotApplyOutsideItsRange(t *testing.T) {
	model := annotationTestModel()
	annotation := model.addAnnotation(Annotation{ChangeID: "commit", File: "a.go", NewLines: lineRange{Start: 1, End: 1}, Comment: "before"})
	model.focusedAnnotationID = annotation.ID
	model.cursor = 1

	_, handled := model.HandleIntent(intents.AnnotationDelete{})
	assert.True(t, handled)
	assert.Len(t, model.annotations.All(), 1)
	_, handled = model.HandleIntent(intents.AnnotationAdd{})
	assert.True(t, handled)
	assert.True(t, model.editing)
	assert.Zero(t, model.focusedAnnotationID)
}

func TestMovingAwayClearsFocusedAnnotation(t *testing.T) {
	model := annotationTestModel()
	annotation := model.addAnnotation(Annotation{ChangeID: "commit", File: "a.go", NewLines: lineRange{Start: 1, End: 1}, Comment: "before"})
	model.focusedAnnotationID = annotation.ID

	model.moveCursor(1, false)

	assert.Zero(t, model.focusedAnnotationID)
}

func TestFocusedAnnotationRemainsActiveAcrossItsRange(t *testing.T) {
	model := annotationTestModel()
	annotation := model.addAnnotation(Annotation{ChangeID: "commit", File: "a.go", NewLines: lineRange{Start: 1, End: 2}, Comment: "before"})
	model.focusedAnnotationID = annotation.ID

	model.moveCursor(1, false)

	assert.Equal(t, annotation.ID, model.focusedAnnotationID)
	_, handled := model.HandleIntent(intents.AnnotationDelete{})
	assert.True(t, handled)
	assert.Empty(t, model.annotations.All())
}

func TestDeleteFindsSingleAnnotationAtCursorWithoutPickerFocus(t *testing.T) {
	model := annotationTestModel()
	model.addAnnotation(Annotation{ChangeID: "commit", File: "a.go", NewLines: lineRange{Start: 1, End: 2}, Comment: "before"})
	model.cursor = 1

	_, handled := model.HandleIntent(intents.AnnotationDelete{})

	assert.True(t, handled)
	assert.Empty(t, model.annotations.All())
}

func annotationTestModel() *Model {
	file := patchFile{NewPath: "a.go", Lines: []patchLine{
		{Kind: lineContext, Raw: " one", Content: "one", OldLine: 1, NewLine: 1},
		{Kind: lineContext, Raw: " two", Content: "two", OldLine: 2, NewLine: 2},
	}}
	return &Model{
		document: reviewDocument{
			revision: "commit",
			files:    []fileItem{{Path: "a.go", Patch: &file}},
		},
	}
}

func TestClearAnnotationsOnlyOnExplicitAction(t *testing.T) {
	model := New(nil, "")
	model.addAnnotation(Annotation{ChangeID: "one", File: "a.go", Comment: "one"})
	model.addAnnotation(Annotation{ChangeID: "two", File: "b.go", Comment: "two"})

	_, handled := model.HandleIntent(intents.AnnotationClear{})

	assert.True(t, handled)
	assert.Empty(t, model.annotations.All())
}

func TestCopyAnnotationsCopiesAllAnnotationsAndShowsConfirmation(t *testing.T) {
	model := New(nil, "")
	model.addAnnotation(Annotation{
		File:     "a.go",
		NewLines: lineRange{Start: 2, End: 2},
		Snippet:  "+new",
		Comment:  "note",
	})

	cmd, handled := model.HandleIntent(intents.AnnotationCopy{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	commands, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)

	var copied, confirmed bool
	for _, command := range commands {
		message := command()
		copied = copied || fmt.Sprint(message) == formatAnnotationsMarkdown(model.annotations.All())
		if flash, ok := message.(intents.AddMessage); ok {
			confirmed = flash.Text == "Copied 1 annotation"
		}
	}
	assert.True(t, copied)
	assert.True(t, confirmed)
}

func TestCopyAnnotationsReportsNoAnnotations(t *testing.T) {
	model := New(nil, "")

	cmd, handled := model.HandleIntent(intents.AnnotationCopy{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	message, ok := cmd().(intents.AddMessage)
	require.True(t, ok)
	assert.Equal(t, "No annotations to copy", message.Text)
}

func TestFullFileAnnotationUsesDiffContextSnippet(t *testing.T) {
	model := &Model{
		document: reviewDocument{
			revision:     "commit",
			files:        []fileItem{{Path: "a.go", Content: []string{"one", "two"}}},
			presentation: filePresentation,
		},
		cursor:          1,
		selectionAnchor: 0,
	}

	annotation := model.newAnnotation("note")

	assert.Equal(t, " one\n two", annotation.Snippet)
}

func TestMouseClickMovesCursorToVisibleSourceLine(t *testing.T) {
	model := mouseTestModel(8)
	display := render.NewDisplayContext()
	model.ViewRect(display, layout.NewBox(layout.Rect(0, 0, 40, 6)))

	message, handled := display.ProcessMouseEvent(tea.MouseClickMsg{
		X:      15,
		Y:      3,
		Button: tea.MouseLeft,
	})
	require.True(t, handled)
	assert.Equal(t, lineClickedMsg{SourceIndex: 2}, message)

	model.Update(message)
	assert.Equal(t, 2, model.cursor)
	assert.Equal(t, -1, model.selectionAnchor)
}

func TestMouseWheelScrollsAnnotationViewport(t *testing.T) {
	model := mouseTestModel(10)
	display := render.NewDisplayContext()
	model.ViewRect(display, layout.NewBox(layout.Rect(0, 0, 40, 5)))

	message, handled := display.ProcessMouseEvent(tea.MouseWheelMsg{
		X:      15,
		Y:      3,
		Button: tea.MouseWheelDown,
	})
	require.True(t, handled)
	assert.Equal(t, scrollMsg{Delta: 3}, message)

	model.Update(message)
	assert.Equal(t, 3, model.scrollY)

	message, handled = display.ProcessMouseEvent(tea.MouseWheelMsg{
		X:      15,
		Y:      3,
		Button: tea.MouseWheelRight,
	})
	require.True(t, handled)
	assert.Equal(t, scrollMsg{Delta: 3, Horizontal: true}, message)

	model.Update(message)
	assert.Equal(t, 3, model.scrollX)
}

func TestWrappedPatchLinePaintsVisibleRowsDirectly(t *testing.T) {
	model := mouseTestModel(1)
	model.document.files[0].Patch.Lines[0].Content = "abcdefghijklmno"
	model.wrap = true
	model.scrollY = 1

	display := render.NewDisplayContext()
	model.ViewRect(display, layout.NewBox(layout.Rect(0, 0, 17, 3)))
	rendered := display.RenderToString(17, 3)
	rows := strings.Split(ansi.Strip(rendered), "\n")
	require.Len(t, rows, 3)
	assert.Equal(t, "          │ fghij", strings.TrimRight(rows[1], " "))
	assert.Equal(t, "          │ klmno", strings.TrimRight(rows[2], " "))

	message, handled := display.ProcessMouseEvent(tea.MouseClickMsg{
		X:      15,
		Y:      1,
		Button: tea.MouseLeft,
	})
	require.True(t, handled)
	assert.Equal(t, lineClickedMsg{SourceIndex: 0}, message)
}

func TestHorizontalScrollOnlySlicesPatchContent(t *testing.T) {
	model := mouseTestModel(1)
	model.document.files[0].Patch.Lines[0].Content = "abcdefghij"
	model.scrollX = 5

	rendered := ansi.Strip(test.RenderImmediate(model, 17, 2))
	rows := strings.Split(rendered, "\n")
	require.Len(t, rows, 2)
	assert.Equal(t, "   1    1 │ fghij", strings.TrimRight(rows[1], " "))
}

func mouseTestModel(lineCount int) *Model {
	lines := make([]patchLine, lineCount)
	for index := range lines {
		lines[index] = patchLine{
			Kind:    lineContext,
			Raw:     fmt.Sprintf(" line %d", index+1),
			Content: fmt.Sprintf("line %d", index+1),
			OldLine: index + 1,
			NewLine: index + 1,
		}
	}
	file := patchFile{NewPath: "a.go", Lines: lines}
	return &Model{
		document: reviewDocument{
			revision: "commit",
			files:    []fileItem{{Path: "a.go", Patch: &file}},
		},
		selectionAnchor: -1,
	}
}

func TestDiffSelectionSkipsHeadersAndCreatesSideAwareRange(t *testing.T) {
	file := patchFile{
		NewPath: "a.go",
		Lines: []patchLine{
			{Kind: lineHunk, Raw: "@@ -10,2 +10,2 @@", Content: "@@ -10,2 +10,2 @@"},
			{Kind: lineContext, Raw: " same", Content: "same", OldLine: 10, NewLine: 10},
			{Kind: lineRemoved, Raw: "-old", Content: "old", OldLine: 11},
			{Kind: lineAdded, Raw: "+new", Content: "new", NewLine: 11},
		},
	}
	model := &Model{
		document: reviewDocument{
			revision: "commit",
			files:    []fileItem{{Path: "a.go", Patch: &file}},
		},
		selectionAnchor: -1,
		cursor:          1,
	}

	model.moveCursor(1, true)
	annotation := model.newAnnotation("note")

	assert.Equal(t, 2, model.cursor)
	assert.Equal(t, lineRange{Start: 10, End: 11}, annotation.OldLines)
	assert.Equal(t, lineRange{Start: 10, End: 10}, annotation.NewLines)
	assert.Equal(t, " same\n-old", annotation.Snippet)
}
