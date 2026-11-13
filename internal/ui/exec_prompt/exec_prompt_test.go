package exec_prompt

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func newContext(t *testing.T) *context.MainContext {
	// keep the user's history files out of the tests
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	return &context.MainContext{Histories: config.NewHistories()}
}

func collect(model *Model, cmd tea.Cmd) []tea.Msg {
	var msgs []tea.Msg
	test.SimulateModel(model, cmd, func(msg tea.Msg) { msgs = append(msgs, msg) })
	return msgs
}

func TestNew_LoadsHistoryForMode(t *testing.T) {
	ctx := newContext(t)
	ctx.Histories.GetHistory(config.HistoryKey("exec sh"), true).Append("ls -la")
	ctx.Histories.GetHistory(config.HistoryKey("exec jj"), true).Append("log -n1")

	m := New(ctx, common.ExecShell, "")

	assert.Equal(t, 1, m.fuzzy.Len())
	assert.Equal(t, "ls -la", m.fuzzy.String(0))
	assert.False(t, m.input.ShowSuggestions, "history is shown in the suggestion list, not inline")
}

func TestApply_SendsExecMsgAndSavesHistory(t *testing.T) {
	ctx := newContext(t)
	m := New(ctx, common.ExecJJ, "")

	collect(m, test.Type("log -n1"))
	msgs := collect(m, func() tea.Msg { return intents.Apply{} })

	assert.Contains(t, msgs, common.CloseViewMsg{Applied: true})
	assert.Contains(t, msgs, common.ExecMsg{Line: "log -n1", Mode: common.ExecJJ})
	assert.Equal(t, config.HistoryEntries{"log -n1"}, ctx.Histories.GetHistory(config.HistoryKey("exec jj"), true).Entries())
}

func TestApply_UsesSelectedSuggestion(t *testing.T) {
	previous := config.Current.Suggest.Exec.Mode
	t.Cleanup(func() { config.Current.Suggest.Exec.Mode = previous })
	config.Current.Suggest.Exec.Mode = "fuzzy"

	ctx := newContext(t)
	ctx.Histories.GetHistory(config.HistoryKey("exec sh"), true).Append("echo 'quoted'")
	m := New(ctx, common.ExecShell, "")

	collect(m, test.Type("echo"))
	msgs := collect(m, func() tea.Msg { return intents.Apply{} })

	assert.Contains(t, msgs, common.ExecMsg{Line: "echo 'quoted'", Mode: common.ExecShell})
}

func TestApply_EmptyInputIsNotSaved(t *testing.T) {
	ctx := newContext(t)
	m := New(ctx, common.ExecJJ, "")

	collect(m, func() tea.Msg { return intents.Apply{} })

	assert.Empty(t, ctx.Histories.GetHistory(config.HistoryKey("exec jj"), true).Entries())
}

func TestCancel_Closes(t *testing.T) {
	m := New(newContext(t), common.ExecJJ, "")

	msgs := collect(m, func() tea.Msg { return intents.Cancel{} })

	assert.Contains(t, msgs, common.CloseViewMsg{})
}

func TestNew_PrefillsFailedLine(t *testing.T) {
	ctx := newContext(t)
	ctx.Histories.GetHistory(config.HistoryKey("exec sh"), true).Append("invalid commands")

	previous := config.Current.Suggest.Exec.Mode
	t.Cleanup(func() { config.Current.Suggest.Exec.Mode = previous })
	config.Current.Suggest.Exec.Mode = "fuzzy"

	m := New(ctx, common.ExecShell, "invalid command")

	assert.Equal(t, "$ ", m.input.Prompt)
	assert.Equal(t, "invalid command", m.input.Value())
	assert.Equal(t, len("invalid command"), m.input.Position())
	assert.Equal(t, "invalid commands", m.fuzzy.Selected(), "suggestions are searched for the prefilled line")
}

func TestViewRect_DrawsLabelAndInputAboveStatusRow(t *testing.T) {
	m := New(newContext(t), common.ExecShell, "")

	dl := render.NewDisplayContext()
	m.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 3)))
	rendered := dl.RenderToString(40, 3)

	assert.Contains(t, rendered, "exec sh")
	assert.Contains(t, rendered, "$ ")
	assert.Equal(t, 28, m.input.Width())
	if assert.NotNil(t, dl.Cursor()) {
		assert.Equal(t, 1, dl.Cursor().Position.Y)
	}
}
