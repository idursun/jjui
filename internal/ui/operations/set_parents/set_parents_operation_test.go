package set_parents

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/require"
)

func TestApplyKeepsExistingParentOrderAndAppendsAddedInSelectionOrder(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetParents("target")).SetOutput([]byte("p1 p2 p3"))
	commandRunner.Expect(jj.SetParents("target", []string{"p1", "p3", "b", "a"}))
	defer commandRunner.Verify()

	target := &jj.Commit{ChangeId: "target", CommitId: "target"}
	op := NewModel(test.NewTestContext(commandRunner), target, target)

	toggle := func(changeId, commitId string) {
		op.Update(common.SelectionChangedMsg{Item: common.SelectedRevision{ChangeId: changeId, CommitId: commitId}})
		_, handled := op.HandleIntent(intents.SetParentsToggleSelect{})
		require.True(t, handled)
	}
	toggle("p2", "p2-commit") // remove an existing parent
	toggle("b", "b-commit")   // add b, then a
	toggle("a", "a-commit")

	cmd, handled := op.HandleIntent(intents.Apply{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	test.SimulateModel(op, cmd, func(tea.Msg) {})
}
