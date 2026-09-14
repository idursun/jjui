package describe

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelUnchangedDescriptionClosesImmediately(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change")).SetOutput([]byte("original"))
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), &jj.Commit{ChangeId: "change", CommitId: "commit"})

	cmd, handled := op.HandleIntent(intents.Cancel{})

	require.True(t, handled)
	require.NotNil(t, cmd)
	assert.IsType(t, common.CloseViewMsg{}, cmd())
	assert.Nil(t, op.confirmation)
}

func TestCancelChangedDescriptionShowsConfirmation(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change")).SetOutput([]byte("original"))
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), &jj.Commit{ChangeId: "change", CommitId: "commit"})
	op.input.SetValue("changed")

	cmd, handled := op.HandleIntent(intents.Cancel{})

	assert.True(t, handled)
	assert.Nil(t, cmd)
	require.NotNil(t, op.confirmation)
	require.NotEmpty(t, op.Scopes())
	assert.Equal(t, keybindings.ScopeName(actions.ScopeInlineDescribeConfirmation), op.Scopes()[0].Name)
	assert.Equal(t, op.confirmation.Styles.Border.GetBackground(), op.confirmation.Styles.Text.GetBackground())
	assert.Equal(t, op.confirmation.Styles.Border.GetBackground(), op.confirmation.Styles.Dimmed.GetBackground())
	assert.Contains(t, test.Stripped(test.RenderImmediate(op, 60, 10)), "You have unsaved changes. Discard them?")
}

func TestCancelConfirmationKeepsEmptyDescriptionDraft(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change")).SetOutput(nil)
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), &jj.Commit{ChangeId: "change", CommitId: "commit"})
	op.input.SetValue("draft")
	_, _ = op.HandleIntent(intents.Cancel{})

	cmd, handled := op.HandleIntent(intents.Cancel{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	op.Update(cmd())

	assert.Equal(t, "draft", op.input.Value())
}

func TestCancelConfirmationKeepsDraft(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change")).SetOutput([]byte("original"))
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), &jj.Commit{ChangeId: "change", CommitId: "commit"})
	op.input.SetValue("changed")
	_, _ = op.HandleIntent(intents.Cancel{})

	cmd, handled := op.HandleIntent(intents.Cancel{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	assert.IsType(t, confirmation.CloseMsg{}, cmd())
	op.Update(confirmation.CloseMsg{})

	assert.Nil(t, op.confirmation)
	assert.Equal(t, "changed", op.input.Value())
}

func TestDiscardConfirmationClosesEditor(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change")).SetOutput([]byte("original"))
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), &jj.Commit{ChangeId: "change", CommitId: "commit"})
	op.input.SetValue("changed")
	_, _ = op.HandleIntent(intents.Cancel{})

	cmd := op.Update(tea.KeyPressMsg{Code: 'y'})

	require.NotNil(t, cmd)
	assert.IsType(t, common.CloseViewMsg{}, cmd())
}

func TestEmbeddedHeight_UsesDynamicHeight(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change")).SetOutput([]byte("this description should wrap onto multiple lines"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	op := NewOperation(ctx, &jj.Commit{ChangeId: "change", CommitId: "commit"})

	height := op.EmbeddedHeight(
		&jj.Commit{ChangeId: "change", CommitId: "commit"},
		operations.RenderOverDescription,
		12,
	)

	assert.Greater(t, height, 1)
}

func TestViewRect_SyncsInputSizeForWrappedCursorMovement(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change")).SetOutput([]byte("one two three four five six"))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	op := NewOperation(ctx, &jj.Commit{ChangeId: "change", CommitId: "commit"})
	op.input.MoveToBegin()

	dl := render.NewDisplayContext()
	op.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 8, 2)))

	assert.Equal(t, 8, op.input.Width())
	assert.Equal(t, 2, op.input.Height())

	op.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	assert.Equal(t, 0, op.input.Line())
	assert.Equal(t, 1, op.input.LineInfo().RowOffset)
}

func TestCtrlUYanksTextWithCtrlY(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change")).SetOutput([]byte("hello world"))
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), &jj.Commit{ChangeId: "change", CommitId: "commit"})
	op.input.MoveToEnd()

	op.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	assert.Empty(t, op.input.Value())

	op.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	assert.Equal(t, "hello world", op.input.Value())
}

func TestCtrlUYanksNewlineAtLineStart(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change")).SetOutput([]byte("hello\nworld"))
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), &jj.Commit{ChangeId: "change", CommitId: "commit"})
	op.input.MoveToEnd()
	op.input.CursorStart()
	op.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	assert.Equal(t, "helloworld", op.input.Value())

	op.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	assert.Equal(t, "hello\nworld", op.input.Value())
}

func TestCtrlKYanksTextWithCtrlY(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change")).SetOutput([]byte("hello world"))
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), &jj.Commit{ChangeId: "change", CommitId: "commit"})
	op.input.SetCursorColumn(6)

	op.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	assert.Equal(t, "hello ", op.input.Value())

	op.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	assert.Equal(t, "hello world", op.input.Value())
}

func TestCtrlKYanksNewlineAtLineEnd(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change")).SetOutput([]byte("hello\nworld"))
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), &jj.Commit{ChangeId: "change", CommitId: "commit"})
	op.input.MoveToEnd()
	op.input.CursorUp()
	op.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	assert.Equal(t, "helloworld", op.input.Value())

	op.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	assert.Equal(t, "hello\nworld", op.input.Value())
}

func TestClearYanksTextWithCtrlY(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription("change")).SetOutput([]byte("hello\nworld"))
	defer commandRunner.Verify()

	op := NewOperation(test.NewTestContext(commandRunner), &jj.Commit{ChangeId: "change", CommitId: "commit"})

	op.Update(intents.InlineDescribeClear{})
	assert.Empty(t, op.input.Value())

	op.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	assert.Equal(t, "hello\nworld", op.input.Value())
}
