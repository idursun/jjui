package revisions

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
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
	model.opStack = []operations.Operation{operations.NewDefault(), base, child}

	_ = test.RenderImmediate(model, 100, 20)

	assert.Zero(t, base.viewRectCalls)
	assert.Equal(t, 1, child.viewRectCalls)
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
