package describe

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

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
