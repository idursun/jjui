package flash

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
)

func TestAdd_IgnoresEmptyMessages(t *testing.T) {
	m := New()

	id := m.add("   ", nil)

	assert.Zero(t, id)
	assert.Empty(t, m.messages)
}

func TestUpdate_AddsSuccessMessageAndSchedulesExpiry(t *testing.T) {
	m := New()

	cmd := m.Update(common.CommandCompletedMsg{Output: "  success  ", Err: nil})

	assert.NotNil(t, cmd)
	if assert.Len(t, m.messages, 1) {
		assert.Equal(t, "success", m.messages[0].text)
		assert.Nil(t, m.messages[0].error)
	}
	assert.Empty(t, m.messageHistory)
}

func TestUpdate_AddsErrorMessageWithoutExpiry(t *testing.T) {
	m := New()

	cmd := m.Update(common.CommandCompletedMsg{Output: "", Err: errors.New("boom")})

	assert.Nil(t, cmd)
	if assert.Len(t, m.messages, 1) {
		assert.EqualError(t, m.messages[0].error, "boom")
		assert.Equal(t, "", m.messages[0].text)
	}
	assert.Empty(t, m.messageHistory)
}

func TestUpdate_ExpiresMessages(t *testing.T) {
	m := New()

	first := m.add("first", nil)
	m.add("second", nil)

	m.Update(expireMessageMsg{id: first})

	if assert.Len(t, m.messages, 1) {
		assert.Equal(t, "second", m.messages[0].text)
	}
	assert.Empty(t, m.messageHistory)
}

func TestView_StacksFromBottomRight(t *testing.T) {
	m := New()

	m.add("abc", nil)
	m.add("de", nil)

	dl := render.NewDisplayContext()
	box := layout.NewBox(layout.Rect(0, 0, 30, 12))
	m.ViewRect(dl, box)
	rendered := strings.Split(dl.RenderToString(box.R.Dx(), box.R.Dy()), "\n")

	abcY := -1
	deY := -1
	for i, line := range rendered {
		if strings.Contains(line, "abc") {
			abcY = i
		}
		if strings.Contains(line, "de") {
			deY = i
		}
	}

	if assert.NotEqual(t, -1, abcY) && assert.NotEqual(t, -1, deY) {
		assert.Greater(t, abcY, deY, "newer flash messages should stack lower on screen")
	}
}

func TestDeleteOldest_RemovesFirstMessage(t *testing.T) {
	m := New()

	m.add("first", nil)
	m.add("second", nil)
	assert.True(t, m.Any())

	m.DeleteOldest()

	if assert.Len(t, m.messages, 1) {
		assert.Equal(t, "second", m.messages[0].text)
	}
}

func TestHistory_CompletionBeforeRunning_ReconcilesByCommandID(t *testing.T) {
	m := New()

	cmd := m.Update(common.CommandCompletedMsg{ID: 9, Output: "ok", Err: nil})
	assert.Nil(t, cmd)
	assert.Empty(t, m.commandHistorySnapshot())
	assert.Equal(t, 0, m.LiveMessagesCount())

	cmd = m.Update(common.CommandRunningMsg{ID: 9, Command: "jj git fetch"})
	assert.NotNil(t, cmd)
	snapshot := m.commandHistorySnapshot()
	if assert.Len(t, snapshot, 1) {
		assert.Equal(t, "jj git fetch", snapshot[0].Command)
		assert.Equal(t, "ok", snapshot[0].Text)
	}
}

func TestHistory_IsBoundedToConfiguredLimit(t *testing.T) {
	m := New()

	for i := 1; i <= HistoryLimit+5; i++ {
		m.AddWithCommand(fmt.Sprintf("m-%d", i), fmt.Sprintf("jj cmd %d", i), nil)
	}

	snapshot := m.commandHistorySnapshot()
	if assert.Len(t, snapshot, HistoryLimit) {
		assert.Equal(t, "m-6", snapshot[0].Text)
		assert.Equal(t, "m-55", snapshot[len(snapshot)-1].Text)
	}
}

func TestHistory_OnlyContainsCommandMessages(t *testing.T) {
	m := New()

	m.Update(intents.AddMessage{Text: "user message"})
	m.AddWithCommand("command output", "jj status", nil)

	snapshot := m.commandHistorySnapshot()
	if assert.Len(t, snapshot, 1) {
		assert.Equal(t, "jj status", snapshot[0].Command)
		assert.Equal(t, "command output", snapshot[0].Text)
	}
}
