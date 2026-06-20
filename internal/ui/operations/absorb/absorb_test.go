package absorb

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

var source = &jj.Commit{ChangeId: "c"}

func newTestOperation(t *testing.T, runner *test.CommandRunner) *Operation {
	t.Helper()
	runner.Expect(jj.AbsorbDefaultTargets("c")).SetOutput([]byte("a\nb\n"))
	return NewOperation(test.NewTestContext(runner), source, source)
}

func selectRevision(op *Operation, commit *jj.Commit) {
	op.Update(common.SelectionChangedMsg{
		Item: common.SelectedRevision{
			ChangeId: commit.GetChangeId(),
			CommitId: commit.CommitId,
		},
	})
}

func Test_DefaultsLoadedIntoTargetsAndDefaults(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	op := newTestOperation(t, commandRunner)

	assert.True(t, op.targets["a"])
	assert.True(t, op.targets["b"])
	assert.True(t, op.defaults["a"])
	assert.True(t, op.defaults["b"])
}

func Test_AcceptWithDefaults_OmitsInto(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Absorb("c", nil))
	defer commandRunner.Verify()

	op := newTestOperation(t, commandRunner)
	test.SimulateModel(op, op.Init())

	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func Test_ToggleOff_PassesRemainingTargets(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Absorb("c", []string{"a"}))
	defer commandRunner.Verify()

	op := newTestOperation(t, commandRunner)
	test.SimulateModel(op, op.Init())

	selectRevision(op, &jj.Commit{ChangeId: "b"})
	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbToggleSelect{} })

	assert.False(t, op.targets["b"])
	assert.True(t, op.targets["a"])

	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func Test_ToggleOnExtra_AddsToTargets(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Absorb("c", []string{"a", "b", "z"}))
	defer commandRunner.Verify()

	op := newTestOperation(t, commandRunner)
	test.SimulateModel(op, op.Init())

	selectRevision(op, &jj.Commit{ChangeId: "z"})
	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbToggleSelect{} })

	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func Test_SelectDescendants_ReplacesTargetsWithCurrentToSourceSegment(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.AbsorbDefaultTargets("c")).SetOutput([]byte("x\ny\n"))
	commandRunner.Expect(jj.GetIdsFromRevset("mutable() & (a:: & ::c)")).SetOutput([]byte("a\nb\nc\n"))
	commandRunner.Expect(jj.Absorb("c", []string{"a", "b"}))
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), source, source)
	test.SimulateModel(op, op.Init())

	selectRevision(op, &jj.Commit{ChangeId: "a"})
	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbSelectDescendants{} })

	assert.True(t, op.targets["a"])
	assert.True(t, op.targets["b"])
	assert.False(t, op.targets["c"])
	assert.False(t, op.targets["x"])
	assert.False(t, op.targets["y"])

	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func Test_SelectDescendants_EmptyResultClearsTargets(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.AbsorbDefaultTargets("c")).SetOutput([]byte("x\ny\n"))
	commandRunner.Expect(jj.GetIdsFromRevset("mutable() & (a:: & ::c)")).SetOutput([]byte(""))
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), source, source)
	test.SimulateModel(op, op.Init())

	selectRevision(op, &jj.Commit{ChangeId: "a"})
	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbSelectDescendants{} })

	assert.Empty(t, op.targets)

	var msgs []tea.Msg
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} }, func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})
	assert.Contains(t, msgs, common.CloseViewMsg{})
	assert.NotContains(t, msgs, common.CloseViewMsg{Applied: true})
}

func Test_ToggleOnThenOff_RestoresDefaultsAndOmitsInto(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Absorb("c", nil))
	defer commandRunner.Verify()

	op := newTestOperation(t, commandRunner)
	test.SimulateModel(op, op.Init())

	selectRevision(op, &jj.Commit{ChangeId: "z"})
	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbToggleSelect{} })
	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbToggleSelect{} })

	assert.False(t, op.targets["z"])

	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func Test_ToggleOffThenOn_RestoresDefaultsAndOmitsInto(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Absorb("c", nil))
	defer commandRunner.Verify()

	op := newTestOperation(t, commandRunner)
	test.SimulateModel(op, op.Init())

	selectRevision(op, &jj.Commit{ChangeId: "a"})
	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbToggleSelect{} })
	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbToggleSelect{} })

	assert.True(t, op.targets["a"])
	assert.True(t, op.targets["b"])

	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func Test_EmptyDefaultsToggleThenOff_OmitsInto(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.AbsorbDefaultTargets("c")).SetOutput([]byte(""))
	commandRunner.Expect(jj.Absorb("c", nil))
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), source, source)
	test.SimulateModel(op, op.Init())

	selectRevision(op, &jj.Commit{ChangeId: "z"})
	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbToggleSelect{} })
	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbToggleSelect{} })

	assert.Empty(t, op.targets)
	assert.Empty(t, op.defaults)

	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func Test_AcceptWithEmptyTargets_ClosesWithoutRunning(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	op := newTestOperation(t, commandRunner)
	test.SimulateModel(op, op.Init())

	selectRevision(op, &jj.Commit{ChangeId: "a"})
	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbToggleSelect{} })
	selectRevision(op, &jj.Commit{ChangeId: "b"})
	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbToggleSelect{} })

	assert.Empty(t, op.targets)

	var msgs []tea.Msg
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} }, func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})
	assert.Contains(t, msgs, common.CloseViewMsg{})
	assert.NotContains(t, msgs, common.CloseViewMsg{Applied: true})
}

func Test_ToggleSourceIsNoOp(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	op := newTestOperation(t, commandRunner)
	test.SimulateModel(op, op.Init())

	test.SimulateModel(op, func() tea.Msg { return intents.AbsorbToggleSelect{} })

	assert.False(t, op.targets[source.GetChangeId()])
}

func Test_RenderShowsSourceMarker(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	op := newTestOperation(t, commandRunner)
	out := op.Render(source, operations.RenderBeforeChangeId)
	assert.Contains(t, out, "<< absorb >>")
}

func Test_Cancel(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()

	op := newTestOperation(t, commandRunner)
	test.SimulateModel(op, op.Init())

	var msgs []tea.Msg
	test.SimulateModel(op, func() tea.Msg { return intents.Cancel{} }, func(msg tea.Msg) {
		msgs = append(msgs, msg)
	})
	assert.Contains(t, msgs, common.CloseViewMsg{})
}
