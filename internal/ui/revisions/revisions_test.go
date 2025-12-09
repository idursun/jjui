package revisions

import (
	"testing"

	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
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
`, err: nil}
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
				Segments: []*screen.Segment{{Text: "a"}}},
		},
	},
	{
		Commit: &jj.Commit{ChangeId: "b", CommitId: "9"},
		Lines: []*parser.GraphRowLine{
			{
				Gutter:   parser.GraphGutter{Segments: []*screen.Segment{{Text: "|"}}},
				Segments: []*screen.Segment{{Text: "b"}}},
		},
	},
}

func TestModel_Navigate(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.SetFrame(cellbuf.Rect(0, 0, 100, 50))
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
			model.SetFrame(cellbuf.Rect(0, 0, 100, 50))
			model.updateGraphRows(rows, "a")
			test.SimulateModel(model, model.Update(tc.intent))
			assert.Contains(t, model.View(), tc.expected)
		})
	}
}
