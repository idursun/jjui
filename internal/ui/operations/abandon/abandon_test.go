package abandon

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

var commit = &jj.Commit{ChangeId: "a"}
var revisions = jj.NewSelectedRevisions(commit)

func Test_Accept(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Abandon(revisions, false))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), revisions)
	test.SimulateModel(model, model.Init())

	model.SetSelectedRevision(commit)
	test.SimulateModel(model, func() tea.Msg { return intents.Apply{} })
}

func Test_Cancel(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), revisions)
	test.SimulateModel(model, model.Init())

	model.SetSelectedRevision(commit)

	test.SimulateModel(model, func() tea.Msg { return intents.Cancel{} })
}

func Test_SelectDescendantsToggleClearsSelection(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetIdsFromRevset("c::")).SetOutput([]byte("c\nd\n"))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), jj.NewSelectedRevisions(&jj.Commit{ChangeId: "c"}))
	test.SimulateModel(model, model.Init())

	model.SetSelectedRevision(&jj.Commit{ChangeId: "c"})
	test.SimulateModel(model, func() tea.Msg { return intents.AbandonSelectDescendants{} })
	test.SimulateModel(model, func() tea.Msg { return intents.AbandonSelectDescendants{} })

	assert.Empty(t, model.selectedRevisions.GetIds())
}

func Test_SelectDescendantsReplacesRevisionSelection(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetIdsFromRevset("c::")).SetOutput([]byte("c\nd\n"))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), jj.NewSelectedRevisions(&jj.Commit{ChangeId: "c"}))
	test.SimulateModel(model, model.Init())

	model.SetSelectedRevision(&jj.Commit{ChangeId: "c"})
	test.SimulateModel(model, func() tea.Msg { return intents.AbandonSelectDescendants{} })

	assert.True(t, model.selections.has("c", selectionTypeDescendants))
	assert.False(t, model.selections.has("c", selectionTypeRevision))
}

func Test_ToggleRevisionReplacesDescendantsSelection(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetIdsFromRevset("c::")).SetOutput([]byte("c\nd\n"))
	commandRunner.Expect(jj.GetIdsFromRevset("c")).SetOutput([]byte("c\n"))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), jj.NewSelectedRevisions(&jj.Commit{ChangeId: "c"}))
	test.SimulateModel(model, model.Init())

	model.SetSelectedRevision(&jj.Commit{ChangeId: "c"})
	test.SimulateModel(model, func() tea.Msg { return intents.AbandonSelectDescendants{} })
	test.SimulateModel(model, func() tea.Msg { return intents.AbandonToggleSelect{} })

	assert.True(t, model.selections.has("c", selectionTypeRevision))
	assert.False(t, model.selections.has("c", selectionTypeDescendants))
	assert.Equal(t, []string{"c"}, model.selectedRevisions.GetIds())
}
