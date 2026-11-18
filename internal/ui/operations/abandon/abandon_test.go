package abandon

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

var commit = &jj.Commit{ChangeId: "a"}
var revisions = jj.NewSelectedRevisions(commit)

func selectRevision(model *Operation, commit *jj.Commit) {
	model.Update(common.SelectionChangedMsg{
		Item: common.SelectedRevision{
			ChangeId: commit.GetChangeId(),
			CommitId: commit.CommitId,
		},
	})
}

func Test_Accept(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.CommandArgs{"abandon", "-r", "a"})
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), revisions, commit)
	test.SimulateModel(model, model.Init())

	test.SimulateModel(model, func() tea.Msg { return intents.Apply{} })
}

func Test_AcceptRetainsBookmarksWhenConfigured(t *testing.T) {
	orig := config.Current.Revisions.Abandon.RetainBookmarks
	t.Cleanup(func() { config.Current.Revisions.Abandon.RetainBookmarks = orig })
	config.Current.Revisions.Abandon.RetainBookmarks = true

	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.CommandArgs{"abandon", "--retain-bookmarks", "-r", "a"})
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), revisions, commit)
	test.SimulateModel(model, model.Init())

	test.SimulateModel(model, func() tea.Msg { return intents.Apply{} })
}

func Test_Cancel(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), revisions, commit)
	test.SimulateModel(model, model.Init())

	test.SimulateModel(model, func() tea.Msg { return intents.Cancel{} })
}

func Test_SelectDescendantsToggleClearsSelection(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetIdsFromRevset("c::")).SetOutput([]byte("c\nd\n"))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), jj.NewSelectedRevisions(&jj.Commit{ChangeId: "c"}), &jj.Commit{ChangeId: "c"})
	test.SimulateModel(model, model.Init())

	test.SimulateModel(model, func() tea.Msg { return intents.AbandonSelectDescendants{} })
	test.SimulateModel(model, func() tea.Msg { return intents.AbandonSelectDescendants{} })

	assert.Empty(t, model.selectedRevisions.GetIds())
}

func Test_SelectDescendantsReplacesRevisionSelection(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetIdsFromRevset("c::")).SetOutput([]byte("c\nd\n"))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), jj.NewSelectedRevisions(&jj.Commit{ChangeId: "c"}), &jj.Commit{ChangeId: "c"})
	test.SimulateModel(model, model.Init())

	test.SimulateModel(model, func() tea.Msg { return intents.AbandonSelectDescendants{} })

	assert.True(t, model.selections.has("c", selectionTypeDescendants))
	assert.False(t, model.selections.has("c", selectionTypeRevision))
}

func Test_ToggleRevisionReplacesDescendantsSelection(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetIdsFromRevset("c::")).SetOutput([]byte("c\nd\n"))
	commandRunner.Expect(jj.GetIdsFromRevset("c")).SetOutput([]byte("c\n"))
	defer commandRunner.Verify()

	model := NewOperation(test.NewTestContext(commandRunner), jj.NewSelectedRevisions(&jj.Commit{ChangeId: "c"}), &jj.Commit{ChangeId: "c"})
	test.SimulateModel(model, model.Init())

	test.SimulateModel(model, func() tea.Msg { return intents.AbandonSelectDescendants{} })
	selectRevision(model, &jj.Commit{ChangeId: "c"})
	test.SimulateModel(model, func() tea.Msg { return intents.AbandonToggleSelect{} })

	assert.True(t, model.selections.has("c", selectionTypeRevision))
	assert.False(t, model.selections.has("c", selectionTypeDescendants))
	assert.Equal(t, []string{"c"}, model.selectedRevisions.GetIds())
}
