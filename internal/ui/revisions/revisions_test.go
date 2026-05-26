package revisions

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	bookmarkops "github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel_highlightChanges(t *testing.T) {
	model := Model{
		rows: []parser.Row{
			{Commit: &jj.Commit{ChangeId: "someother"}},
			{Commit: &jj.Commit{ChangeId: "nyqzpsmt"}},
		},
		output: `
Absorbed changes into these revisions:
  nyqzpsmt 8b1e95e3 change third file
Working copy now at: okrwsxvv 5233c94f (empty) (no description set)
Parent commit      : nyqzpsmt 8b1e95e3 change third file
`, err: nil,
	}
	_ = model.highlightChanges()
	assert.False(t, model.rows[0].IsAffected)
	assert.True(t, model.rows[1].IsAffected)
}

var rows = []parser.Row{
	{
		Commit: &jj.Commit{ChangeId: "a", CommitId: "8"},
		Lines: []*parser.GraphRowLine{
			{
				Gutter:   parser.GraphGutter{Segments: []*screen.Segment{{Text: "|"}}},
				Segments: []*screen.Segment{{Text: "a"}},
				Flags:    parser.Revision,
			},
		},
	},
	{
		Commit: &jj.Commit{ChangeId: "b", CommitId: "9"},
		Lines: []*parser.GraphRowLine{
			{
				Gutter:   parser.GraphGutter{Segments: []*screen.Segment{{Text: "|"}}},
				Segments: []*screen.Segment{{Text: "b"}},
				Flags:    parser.Revision,
			},
		},
	},
}

type viewRectTrackingOp struct {
	name          string
	viewRectCalls int
}

func (o *viewRectTrackingOp) Init() tea.Cmd { return nil }

func (o *viewRectTrackingOp) Update(tea.Msg) tea.Cmd { return nil }

func (o *viewRectTrackingOp) ViewRect(_ *render.DisplayContext, _ layout.Box) {
	o.viewRectCalls++
}

func (o *viewRectTrackingOp) Render(*jj.Commit, operations.RenderPosition) string { return "" }

func (o *viewRectTrackingOp) Name() string { return o.name }

func TestModel_Navigate(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a")

	test.SimulateModel(model, model.Update(intents.Navigate{Delta: 1}))
	assert.Equal(t, "b", model.SelectedRevision().ChangeId)
	test.SimulateModel(model, model.Update(intents.Navigate{Delta: -1}))
	assert.Equal(t, "a", model.SelectedRevision().ChangeId)
}

func TestModel_UpdateGraphRows_DoesNotPrefixMatchImplicitCurrentSelection(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows([]parser.Row{
		{Commit: &jj.Commit{ChangeId: "66d7"}},
		{Commit: &jj.Commit{ChangeId: "66d"}},
	}, "66d7")

	model.updateGraphRows([]parser.Row{
		{Commit: &jj.Commit{ChangeId: "other"}},
		{Commit: &jj.Commit{ChangeId: "66d"}},
	}, "")

	assert.Equal(t, 0, model.Cursor(), "removed current revision must not prefix-match a similarly-starting revision")
	assert.Equal(t, "other", model.SelectedRevision().ChangeId)
}

func TestModel_UpdateGraphRows_DoesNotPrefixMatchExplicitSelection(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)

	model.updateGraphRows([]parser.Row{
		{Commit: &jj.Commit{ChangeId: "66d"}},
		{Commit: &jj.Commit{ChangeId: "other"}},
	}, "66d7")

	assert.Equal(t, 0, model.Cursor(), "explicit selection should fall back to the first row when the exact revision is gone")
	assert.Equal(t, "66d", model.SelectedRevision().ChangeId)
}

func TestModel_NavigateTo(t *testing.T) {
	tests := []struct {
		name         string
		initialRows  []parser.Row
		initial      string
		changeID     string
		resolved     []byte
		resolveErr   error
		wantSelected string
	}{
		{
			name:         "uses exact local change id without resolver",
			initialRows:  rows,
			initial:      "a",
			changeID:     "b",
			wantSelected: "b",
		},
		{
			name:         "uses exact local commit id without resolver",
			initialRows:  rows,
			initial:      "a",
			changeID:     "9",
			wantSelected: "b",
		},
		{
			name:         "resolves full commit id through jj",
			initialRows:  rows,
			initial:      "a",
			changeID:     "full-commit",
			resolved:     []byte("b;9"),
			wantSelected: "b",
		},
		{
			name:         "resolves full change id through jj",
			initialRows:  rows,
			initial:      "a",
			changeID:     "full-change",
			resolved:     []byte("b;9"),
			wantSelected: "b",
		},
		{
			name: "does not use prefix matching",
			initialRows: []parser.Row{
				{Commit: &jj.Commit{ChangeId: "66d", CommitId: "111"}},
				{Commit: &jj.Commit{ChangeId: "other", CommitId: "222"}},
			},
			initial:      "other",
			changeID:     "66d7",
			resolved:     []byte("missing;missing"),
			wantSelected: "other",
		},
		{
			name:         "no move when resolved revision not loaded",
			initialRows:  rows,
			initial:      "a",
			changeID:     "full-change",
			resolved:     []byte("missing;missingcommit"),
			wantSelected: "a",
		},
		{
			name:         "no move when resolver errors",
			initialRows:  rows,
			initial:      "a",
			changeID:     "ambiguous",
			resolveErr:   assert.AnError,
			wantSelected: "a",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			commandRunner := test.NewTestCommandRunner(t)
			if tc.resolved != nil || tc.resolveErr != nil {
				expectation := commandRunner.Expect(jj.ResolveRevisionID(tc.changeID))
				if tc.resolved != nil {
					expectation.SetOutput(tc.resolved)
				}
				if tc.resolveErr != nil {
					expectation.SetError(tc.resolveErr)
				}
			}
			defer commandRunner.Verify()

			ctx := test.NewTestContext(commandRunner)
			model := New(ctx)
			model.updateGraphRows(tc.initialRows, tc.initial)

			test.SimulateModel(model, model.Update(intents.Navigate{ChangeID: tc.changeID}))

			assert.Equal(t, tc.wantSelected, model.SelectedRevision().ChangeId)
		})
	}
}

func TestModel_OpenSquashEmitsSelectionChangedForTargetRevision(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetParent(jj.NewSelectedRevisions(rows[0].Commit))).SetOutput([]byte("9"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.updateGraphRows(rows, "a")
	ctx.SelectedItem = common.SelectedRevision{
		ChangeId: rows[0].Commit.GetChangeId(),
		CommitId: rows[0].Commit.CommitId,
	}

	var gotTargetSelection bool
	cmd := model.Update(intents.OpenSquash{})
	test.SimulateModel(model, cmd, func(msg tea.Msg) {
		selection, ok := msg.(common.SelectionChangedMsg)
		if !ok {
			return
		}
		gotTargetSelection = selection.Item.Equal(common.SelectedRevision{
			ChangeId: rows[1].Commit.GetChangeId(),
			CommitId: rows[1].Commit.CommitId,
		})
	})

	assert.Equal(t, 1, model.Cursor(), "opening squash should move the cursor to the target revision")
	assert.True(t, gotTargetSelection, "opening squash should refresh observers for the target revision")
}

func TestModel_StreamingRefreshFallbackSelectsRevisionChangeID(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "b")
	ctx.SelectedItem = common.SelectedRevision{
		ChangeId: "b",
		CommitId: "9",
	}

	model.internalUpdate(streamingReadyMsg{tag: model.tag.Load()})

	assert.Equal(t, "b", model.revisionToSelect, "streaming refresh should use the stable change id, not the stale commit id")
}

func TestModel_RenderImmediateInNormalMode(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a")

	assert.NotPanics(t, func() {
		rendered := test.RenderImmediate(model, 100, 20)
		assert.Contains(t, rendered, "a")
	})
}

func TestModel_ViewRectOnlyRendersStackedChildren(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a")

	base := &viewRectTrackingOp{name: "base"}
	child := &viewRectTrackingOp{name: "child"}
	model.baseOp = base
	model.layers = []common.ImmediateModel{child}

	_ = test.RenderImmediate(model, 100, 20)

	assert.Zero(t, base.viewRectCalls)
	assert.Equal(t, 1, child.viewRectCalls)
}

func TestModel_ViewRectDoesNotOverwriteEditingOperationCursorWithFocusedObject(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.rows = []parser.Row{textObjectTestRow()}
	model.focusedObjectKind = textObjectGraph
	model.baseOp = bookmarkops.NewSetBookmarkOperation(ctx, "abc", "new-bookmark")

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(5, 3, 80, 8)))

	cursor := dl.Cursor()
	require.NotNil(t, cursor)
	assert.NotEqual(t, 5, cursor.Position.X, "editing cursor should not be overwritten by focused revision object")
}

func TestModel_OperationIntents(t *testing.T) {
	tests := []struct {
		name     string
		intent   intents.Intent
		expected string
	}{
		{
			name:     "abandon",
			intent:   intents.OpenAbandon{},
			expected: "abandon",
		},
		{
			name:     "rebase",
			intent:   intents.OpenRebase{},
			expected: "rebase",
		},
		{
			name:     "duplicate",
			intent:   intents.OpenDuplicate{},
			expected: "duplicate",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := test.NewTestContext(test.NewTestCommandRunner(t))

			model := New(ctx)
			model.updateGraphRows(rows, "a")
			test.SimulateModel(model, model.Update(tc.intent))
			assert.False(t, model.InNormalMode())
			rendered := test.RenderImmediate(model, 100, 50)
			assert.Contains(t, rendered, tc.expected)
		})
	}
}

func TestModel_BookmarkCreateStartsInlineSetBookmarkOnPlaceholder(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovable("a"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.updateGraphRows(rows, "a")
	model.focusedObjectKind = textObjectBookmark

	test.SimulateModel(model, model.Update(intents.BookmarkCreate{}))

	_, ok := model.CurrentOperation().(*bookmarkops.SetBookmarkOperation)
	require.True(t, ok)
}

func TestModel_BookmarkCreateStartsInlineSetBookmarkOnExistingBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovable("abc"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.rows = []parser.Row{textObjectTestRow()}
	model.focusedObjectKind = textObjectBookmark
	model.focusedObjectIndex = 1

	test.SimulateModel(model, model.Update(intents.BookmarkCreate{}))

	_, ok := model.CurrentOperation().(*bookmarkops.SetBookmarkOperation)
	require.True(t, ok)
}

func TestModel_BookmarkRenameStartsInlineRenameForLocalBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("feature;.;false;false;false;def\n"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := modelFocusedOnBookmark(ctx, "feature")

	test.SimulateModel(model, model.Update(intents.BookmarkRename{}))

	_, ok := model.CurrentOperation().(*bookmarkops.RenameBookmarkOperation)
	require.True(t, ok)
}

func TestModel_BookmarkDeleteRunsCommandForLocalBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("feature;.;false;false;false;def\n"))
	commandRunner.Expect(jj.BookmarkDelete("feature"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := modelFocusedOnBookmark(ctx, "feature")

	runFirstBatchCommand(t, model.Update(intents.BookmarkDelete{}))
}

func TestModel_BookmarkTrackRunsCommandForUntrackedRemoteBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("feature;origin;false;false;false;def\n"))
	commandRunner.Expect(jj.BookmarkTrack("feature", "origin"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := modelFocusedOnBookmark(ctx, "feature@origin")

	runFirstBatchCommand(t, model.Update(intents.BookmarkTrack{}))
}

func TestModel_BookmarkUntrackRunsCommandForTrackedRemoteBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("feature;origin;true;false;false;def\n"))
	commandRunner.Expect(jj.BookmarkUntrack("feature", "origin"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := modelFocusedOnBookmark(ctx, "feature@origin")

	runFirstBatchCommand(t, model.Update(intents.BookmarkUntrack{}))
}

func TestModel_BookmarkTrackMatchesRemoteDisplayNamesWithAtSigns(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("feat@team;origin@backup;false;false;false;def\n"))
	commandRunner.Expect(jj.BookmarkTrack("feat@team", "origin@backup"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := modelFocusedOnBookmark(ctx, "feat@team@origin@backup")

	runFirstBatchCommand(t, model.Update(intents.BookmarkTrack{}))
}

func TestModel_InvalidBookmarkActionFlashesWithoutCommand(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("feature;origin;true;false;false;def\n"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := modelFocusedOnBookmark(ctx, "feature@origin")

	var gotFlash bool
	test.SimulateModel(model, model.Update(intents.BookmarkRename{}), func(msg tea.Msg) {
		if flash, ok := msg.(intents.AddMessage); ok {
			gotFlash = flash.Text != ""
		}
	})

	assert.True(t, gotFlash)
}

func modelFocusedOnBookmark(ctx *appContext.MainContext, value string) *Model {
	model := New(ctx)
	row := textObjectTestRow()
	row.Lines[0].Segments = []*screen.Segment{
		{Text: "abc", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))},
		{Text: " " + value + " ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("5"))},
		{Text: "def", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))},
	}
	model.rows = []parser.Row{row}
	model.focusedObjectKind = textObjectBookmark
	return model
}

func runFirstBatchCommand(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	require.NotNil(t, cmd)
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	require.True(t, ok, "expected a batch command, got %T", msg)
	require.NotEmpty(t, batch)
	require.NotNil(t, batch[0])
	batch[0]()
}

func TestModel_ForwardsOperationIntentToFocusedOperation(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.updateGraphRows(rows, "a")

	test.SimulateModel(model, model.Update(intents.OpenRebase{}))
	assert.False(t, model.InNormalMode())
	assert.False(t, model.IsEditing())

	test.SimulateModel(model, model.Update(intents.RebaseOpenTargetPicker{}))
	assert.True(t, model.IsEditing(), "rebase target picker should open via dispatched operation intent")
}

func TestModel_TargetPickerCancelClosesEditing(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.updateGraphRows(rows, "a")

	test.SimulateModel(model, model.Update(intents.OpenRebase{}))
	test.SimulateModel(model, model.Update(intents.RebaseOpenTargetPicker{}))
	assert.True(t, model.IsEditing(), "target picker should be editing before cancel")

	test.SimulateModel(model, model.Update(intents.TargetPickerCancel{}))
	assert.False(t, model.IsEditing(), "target picker cancel should exit editing mode")
}
