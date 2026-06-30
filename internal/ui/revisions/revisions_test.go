package revisions

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
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
	name             string
	viewRectCalls    int
	viewRectOrder    *[]string
	selectedRevision *jj.Commit
	selectedFile     string
	selectedCommit   string
}

func (o *viewRectTrackingOp) Init() tea.Cmd { return nil }

func (o *viewRectTrackingOp) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(common.SelectionChangedMsg); ok {
		selected, ok := msg.Item.(common.SelectedRevision)
		if ok {
			o.selectedRevision = &jj.Commit{ChangeId: selected.ChangeId, CommitId: selected.CommitId}
		}
	}
	return nil
}

func (o *viewRectTrackingOp) ViewRect(_ *render.DisplayContext, _ layout.Box) {
	o.viewRectCalls++
	if o.viewRectOrder != nil {
		*o.viewRectOrder = append(*o.viewRectOrder, o.name)
	}
}

func (o *viewRectTrackingOp) Render(*jj.Commit, operations.RenderPosition) string { return "" }

func (o *viewRectTrackingOp) Name() string { return o.name }

func (o *viewRectTrackingOp) Selection() common.SelectionSnapshot {
	if o.selectedCommit != "" {
		return common.SelectionSnapshot{
			Highlighted: common.SelectedCommit{CommitId: o.selectedCommit},
		}
	}
	if o.selectedFile != "" && o.selectedRevision != nil {
		return common.SelectionSnapshot{
			Highlighted: common.SelectedFile{
				ChangeId: o.selectedRevision.GetChangeId(),
				CommitId: o.selectedRevision.CommitId,
				File:     o.selectedFile,
			},
		}
	}
	return common.SelectionSnapshot{}
}

type embeddedClickMsg struct {
	index int
}

type embeddedClickOp struct {
	targetChangeID string
	height         int
}

func (o embeddedClickOp) Init() tea.Cmd { return nil }

func (o embeddedClickOp) Update(tea.Msg) tea.Cmd { return nil }

func (o embeddedClickOp) ViewRect(dl *render.DisplayContext, box layout.Box) {
	for i := range o.height {
		index := i
		dl.AddInteractionFn(
			layout.Rect(box.R.Min.X, box.R.Min.Y+i, box.R.Dx(), 1),
			func(tea.MouseMsg) tea.Msg { return embeddedClickMsg{index: index} },
			render.InteractionClick,
			0,
		)
	}
}

func (o embeddedClickOp) Render(*jj.Commit, operations.RenderPosition) string { return "" }

func (o embeddedClickOp) Name() string { return "embedded click" }

func (o embeddedClickOp) CanEmbed(commit *jj.Commit, pos operations.RenderPosition) bool {
	return commit != nil && commit.GetChangeId() == o.targetChangeID && pos == operations.RenderPositionAfter
}

func (o embeddedClickOp) EmbeddedHeight(commit *jj.Commit, pos operations.RenderPosition, _ int) int {
	if !o.CanEmbed(commit, pos) {
		return 0
	}
	return o.height
}

func TestModel_Navigate(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a", true)

	test.SimulateModel(model, model.Update(intents.Navigate{Delta: 1}))
	assert.Equal(t, "b", model.SelectedRevision().ChangeId)
	test.SimulateModel(model, model.Update(intents.Navigate{Delta: -1}))
	assert.Equal(t, "a", model.SelectedRevision().ChangeId)
}

func TestModel_GoToTopAndBottom(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a")

	test.SimulateModel(model, model.Update(intents.GoToBottom{}))
	assert.Equal(t, len(model.rows)-1, model.Cursor())

	test.SimulateModel(model, model.Update(intents.GoToTop{}))
	assert.Equal(t, 0, model.Cursor())
}

func TestModel_UpdateGraphRows_DoesNotPrefixMatchImplicitCurrentSelection(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows([]parser.Row{
		{Commit: &jj.Commit{ChangeId: "66d7"}},
		{Commit: &jj.Commit{ChangeId: "66d"}},
	}, "66d7", true)

	model.updateGraphRows([]parser.Row{
		{Commit: &jj.Commit{ChangeId: "other"}},
		{Commit: &jj.Commit{ChangeId: "66d"}},
	}, "", true)

	assert.Equal(t, 0, model.Cursor(), "removed current revision must not prefix-match a similarly-starting revision")
	assert.Equal(t, "other", model.SelectedRevision().ChangeId)
}

func TestModel_UpdateGraphRows_DoesNotPrefixMatchExplicitSelection(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)

	model.updateGraphRows([]parser.Row{
		{Commit: &jj.Commit{ChangeId: "66d"}},
		{Commit: &jj.Commit{ChangeId: "other"}},
	}, "66d7", true)

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
			model.updateGraphRows(tc.initialRows, tc.initial, true)

			test.SimulateModel(model, model.Update(intents.Navigate{ChangeID: tc.changeID}))

			assert.Equal(t, tc.wantSelected, model.SelectedRevision().ChangeId)
		})
	}
}

func TestModel_OpenSquashSelectsTargetRevision(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetParent(jj.NewSelectedRevisions(rows[0].Commit))).SetOutput([]byte("9"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.updateGraphRows(rows, "a", true)

	cmd := model.Update(intents.OpenSquash{})
	test.SimulateModel(model, cmd)

	assert.Equal(t, 1, model.Cursor(), "opening squash should move the cursor to the target revision")
	assert.True(t, model.Selection().Highlighted.Equal(common.SelectedRevision{
		ChangeId: rows[1].Commit.GetChangeId(),
		CommitId: rows[1].Commit.CommitId,
	}), "opening squash should update the revisions selection snapshot")
}

func TestModel_UpdateRevisionsPreservesEvologSelection(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.baseOp = &viewRectTrackingOp{name: "evolog", selectedCommit: "9"}
	model.updateGraphRows(rows, "a", true)

	const tag uint64 = 1
	model.tag.Store(tag)
	model.pendingReload = revisionReloadState{tag: tag, selectedRevision: "9"}
	cmd := model.Update(updateRevisionsMsg{rows: rows, tag: tag})
	test.SimulateModel(model, cmd)

	selected, ok := model.Selection().Highlighted.(common.SelectedCommit)
	assert.True(t, ok, "refresh should preserve the evolog selection type while evolog is active")
	assert.Equal(t, "9", selected.CommitId)
}

func TestModel_UpdateRevisionsRefreshesDetailsSelection(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.baseOp = &viewRectTrackingOp{name: "details", selectedFile: "file.txt"}
	model.updateGraphRows(rows, "a", true)

	newRows := []parser.Row{
		{
			Commit: &jj.Commit{ChangeId: "a", CommitId: "10"},
			Lines:  rows[0].Lines,
		},
		{
			Commit: rows[1].Commit,
			Lines:  rows[1].Lines,
		},
	}

	const tag uint64 = 1
	model.tag.Store(tag)
	model.pendingReload = revisionReloadState{tag: tag, selectedRevision: "a"}
	cmd := model.Update(updateRevisionsMsg{rows: newRows, tag: tag})
	test.SimulateModel(model, cmd)

	selected, ok := model.Selection().Highlighted.(common.SelectedFile)
	assert.True(t, ok, "refresh should keep details selection type while details is active")
	assert.Equal(t, "a", selected.ChangeId)
	assert.Equal(t, "10", selected.CommitId)
	assert.Equal(t, "file.txt", selected.File)
}

func TestModel_UpdateRevisionsPreservesViewportOnKeepSelectionRefresh(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a", true)
	model.ensureCursorView = false

	const tag uint64 = 1
	model.tag.Store(tag)
	model.pendingReload = revisionReloadState{tag: tag, selectedRevision: "a", keepSelections: true}
	cmd := model.Update(updateRevisionsMsg{rows: rows, tag: tag})
	test.SimulateModel(model, cmd)

	assert.Equal(t, 0, model.Cursor())
	assert.False(t, model.ensureCursorView, "keep-selection refresh should not request cursor recentering")
}

func TestModel_RenderImmediateInNormalMode(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a", true)

	assert.NotPanics(t, func() {
		rendered := test.RenderImmediate(model, 100, 20)
		assert.Contains(t, rendered, "a")
	})
}

func TestModel_ViewRectRendersBaseOperationAndStackedChildren(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a", true)

	var order []string
	base := &viewRectTrackingOp{name: "base", viewRectOrder: &order}
	child := &viewRectTrackingOp{name: "child", viewRectOrder: &order}
	model.baseOp = base
	model.layers = []common.ImmediateModel{child}

	_ = test.RenderImmediate(model, 100, 20)

	assert.Equal(t, 1, base.viewRectCalls)
	assert.Equal(t, 1, child.viewRectCalls)
	assert.Equal(t, []string{"base", "child"}, order)
}

func TestModel_ViewRectEmbeddedBaseOperationDoesNotRegisterViewportClicks(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "b", true)
	model.baseOp = embeddedClickOp{targetChangeID: "b", height: 4}

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 80, 10)))

	msg, handled := dl.ProcessMouseEvent(tea.MouseClickMsg{
		X:      2,
		Y:      2,
		Button: tea.MouseLeft,
	})
	assert.True(t, handled)
	assert.Equal(t, embeddedClickMsg{index: 0}, msg)
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
			model.updateGraphRows(rows, "a", true)
			test.SimulateModel(model, model.Update(tc.intent))
			assert.False(t, model.InNormalMode())
			rendered := test.RenderImmediate(model, 100, 50)
			assert.Contains(t, rendered, tc.expected)
		})
	}
}

func TestModel_ForwardsOperationIntentToFocusedOperation(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll())
	commandRunner.Expect(jj.TagList())
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.updateGraphRows(rows, "a", true)

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
	model.updateGraphRows(rows, "a", true)

	test.SimulateModel(model, model.Update(intents.OpenRebase{}))
	test.SimulateModel(model, model.Update(intents.RebaseOpenTargetPicker{}))
	assert.True(t, model.IsEditing(), "target picker should be editing before cancel")

	test.SimulateModel(model, model.Update(intents.TargetPickerCancel{}))
	assert.False(t, model.IsEditing(), "target picker cancel should exit editing mode")
}
