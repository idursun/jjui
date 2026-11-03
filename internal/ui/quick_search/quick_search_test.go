package quick_search

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

func TestNew_LoadsSearchHistoryAsInlineSuggestions(t *testing.T) {
	ctx := newContext(t)
	ctx.Histories.GetHistory(config.HistoryKey("search"), true).Append("trunk")

	m := New(ctx)

	assert.True(t, m.input.ShowSuggestions)
	assert.Equal(t, []string{"trunk"}, m.input.AvailableSuggestions())
}

func TestApply_SendsQuickSearchMsgAndSavesHistory(t *testing.T) {
	ctx := newContext(t)
	m := New(ctx)

	collect(m, test.Type("main"))
	msgs := collect(m, func() tea.Msg { return intents.Apply{} })

	assert.Contains(t, msgs, common.CloseViewMsg{Applied: true})
	assert.Contains(t, msgs, common.QuickSearchMsg("main"))
	assert.Equal(t, config.HistoryEntries{"main"}, ctx.Histories.GetHistory(config.HistoryKey("search"), true).Entries())
}

func TestCancel_Closes(t *testing.T) {
	m := New(newContext(t))

	msgs := collect(m, func() tea.Msg { return intents.Cancel{} })

	assert.Contains(t, msgs, common.CloseViewMsg{})
}

func TestViewRect_DrawsLabelAndInputAboveStatusRow(t *testing.T) {
	m := New(newContext(t))

	dl := render.NewDisplayContext()
	m.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 3)))
	rendered := dl.RenderToString(40, 3)

	assert.Contains(t, rendered, "search")
	assert.Contains(t, rendered, "> ")
	if assert.NotNil(t, dl.Cursor()) {
		assert.Equal(t, 1, dl.Cursor().Position.Y)
	}
}
