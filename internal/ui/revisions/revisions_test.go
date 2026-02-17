package revisions

import (
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
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

func TestModel_Navigate(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a")

	test.SimulateModel(model, model.Update(intents.Navigate{Delta: 1}))
	assert.Equal(t, "b", model.SelectedRevision().ChangeId)
	test.SimulateModel(model, model.Update(intents.Navigate{Delta: -1}))
	assert.Equal(t, "a", model.SelectedRevision().ChangeId)
}

func TestModel_OperationIntents(t *testing.T) {
	tests := []struct {
		name     string
		intent   intents.Intent
		expected string
	}{
		{
			name:     "abandon",
			intent:   intents.StartAbandon{},
			expected: "abandon",
		},
		{
			name:     "rebase",
			intent:   intents.StartRebase{},
			expected: "rebase",
		},
		{
			name:     "duplicate",
			intent:   intents.StartDuplicate{},
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

	test.SimulateModel(model, model.Update(intents.StartRebase{}))
	assert.False(t, model.InNormalMode())
	assert.False(t, model.IsEditing())

	test.SimulateModel(model, model.Update(intents.RebaseOpenTargetPicker{}))
	assert.True(t, model.IsEditing(), "rebase target picker should open via dispatched operation intent")
}

func TestModel_StartAceJumpMsg_OpensAceJumpOperation(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(rows, "a")

	test.SimulateModel(model, model.Update(intents.StartAbandon{}))
	assert.Equal(t, "abandon", model.CurrentOperation().Name())

	test.SimulateModel(model, model.Update(common.StartAceJumpMsg{}))
	scopes := model.ScopeChain()
	assert.NotEmpty(t, scopes)
	assert.Equal(t, keybindings.Scope(actions.OwnerAceJump), scopes[0])
	assert.Len(t, scopes, 1)
}
