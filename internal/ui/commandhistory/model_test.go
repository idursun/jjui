package commandhistory

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/flash"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlash_CompletionBeforeRunning_ReconcilesByCommandID(t *testing.T) {
	source := flash.New(test.NewTestContext(test.NewTestCommandRunner(t)))

	cmd := source.Update(common.CommandCompletedMsg{ID: 9, Output: "ok", Err: nil})
	assert.Nil(t, cmd)
	assert.Empty(t, source.CommandHistorySnapshot())
	assert.Equal(t, 0, source.LiveMessagesCount())

	cmd = source.Update(common.CommandRunningMsg{ID: 9, Command: "jj git fetch"})
	assert.NotNil(t, cmd)
	snapshot := source.CommandHistorySnapshot()
	if assert.Len(t, snapshot, 1) {
		assert.Equal(t, "jj git fetch", snapshot[0].Command)
		assert.Equal(t, "ok", snapshot[0].Text)
	}
}

func TestFlash_HistoryIsBoundedToConfiguredLimit(t *testing.T) {
	source := flash.New(test.NewTestContext(test.NewTestCommandRunner(t)))

	for i := 1; i <= flash.HistoryLimit+5; i++ {
		source.AddWithCommand(fmt.Sprintf("m-%d", i), fmt.Sprintf("jj cmd %d", i), nil)
	}

	snapshot := source.CommandHistorySnapshot()
	if assert.Len(t, snapshot, flash.HistoryLimit) {
		assert.Equal(t, "m-6", snapshot[0].Text)
		assert.Equal(t, "m-55", snapshot[len(snapshot)-1].Text)
	}
}

func TestFlash_HistoryOnlyContainsCommandMessages(t *testing.T) {
	source := flash.New(test.NewTestContext(test.NewTestCommandRunner(t)))

	source.Update(intents.AddMessage{Text: "user message"})
	source.AddWithCommand("command output", "jj status", nil)

	snapshot := source.CommandHistorySnapshot()
	if assert.Len(t, snapshot, 1) {
		assert.Equal(t, "jj status", snapshot[0].Command)
		assert.Equal(t, "command output", snapshot[0].Text)
	}
}

func TestCommandHistory_NavigationAdjustsSelection(t *testing.T) {
	source := flash.New(test.NewTestContext(test.NewTestCommandRunner(t)))
	for i := 0; i < 6; i++ {
		source.AddWithCommand(fmt.Sprintf("m-%d", i), fmt.Sprintf("jj cmd %d", i), nil)
	}

	history := New(test.NewTestContext(test.NewTestCommandRunner(t)), source)
	assert.Equal(t, 5, history.selectedIndex)

	history.Update(intents.CommandHistoryNavigate{Delta: 1})
	history.Update(intents.CommandHistoryNavigate{Delta: 1})
	assert.Equal(t, 3, history.selectedIndex)

	history.Update(intents.CommandHistoryNavigate{Delta: -1})
	assert.Equal(t, 4, history.selectedIndex)
}

func TestCommandHistory_ViewOnlyShowsSelectedOutput(t *testing.T) {
	source := flash.New(test.NewTestContext(test.NewTestCommandRunner(t)))
	source.AddWithCommand("older-output", "jj older", nil)
	source.AddWithCommand("newer-output", "jj newer", nil)

	history := New(test.NewTestContext(test.NewTestCommandRunner(t)), source)
	history.Update(intents.CommandHistoryNavigate{Delta: 1}) // select older

	dl := render.NewDisplayContext()
	history.ViewRect(dl, layout.NewBox(cellbuf.Rect(0, 0, 60, 12)))

	var out strings.Builder
	for _, view := range dl.DrawList() {
		out.WriteString(view.Content)
		out.WriteByte('\n')
	}
	rendered := out.String()
	assert.Contains(t, rendered, "jj older")
	assert.Contains(t, rendered, "older-output")
	assert.Contains(t, rendered, "jj newer")
	assert.NotContains(t, rendered, "newer-output")
}

func TestCommandHistory_DeleteSelectedRemovesFromSourceAndLiveMessages(t *testing.T) {
	source := flash.New(test.NewTestContext(test.NewTestCommandRunner(t)))
	source.AddWithCommand("older-output", "jj older", nil)
	source.AddWithCommand("newer-output", "jj newer", nil)

	history := New(test.NewTestContext(test.NewTestCommandRunner(t)), source)
	history.Update(intents.CommandHistoryNavigate{Delta: 1}) // select older
	history.Update(intents.CommandHistoryDeleteSelected{})

	if assert.Len(t, history.items, 1) {
		assert.Equal(t, "jj newer", history.items[0].Command)
	}

	snapshot := source.CommandHistorySnapshot()
	if assert.Len(t, snapshot, 1) {
		assert.Equal(t, "jj newer", snapshot[0].Command)
	}
	assert.Equal(t, 1, source.LiveMessagesCount())
}

func TestCommandHistory_CloseReturnsCloseViewMsg(t *testing.T) {
	source := flash.New(test.NewTestContext(test.NewTestCommandRunner(t)))
	source.AddWithCommand("output", "jj cmd", nil)
	history := New(test.NewTestContext(test.NewTestCommandRunner(t)), source)

	cmd := history.Update(intents.CommandHistoryClose{})
	require.NotNil(t, cmd)
	_, ok := cmd().(common.CloseViewMsg)
	assert.True(t, ok)
}
