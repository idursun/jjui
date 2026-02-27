package flash

import (
	"errors"
	"testing"

	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestAdd_IgnoresEmptyMessages(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))

	id := m.add("   ", nil)

	assert.Zero(t, id)
	assert.Empty(t, m.messages)
}

func TestUpdate_AddsSuccessMessageAndSchedulesExpiry(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))

	cmd := m.Update(common.CommandCompletedMsg{Output: "  success  ", Err: nil})

	assert.NotNil(t, cmd)
	if assert.Len(t, m.messages, 1) {
		assert.Equal(t, "success", m.messages[0].text)
		assert.Nil(t, m.messages[0].error)
	}
	assert.Empty(t, m.messageHistory)
}

func TestUpdate_AddsErrorMessageWithoutExpiry(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))

	cmd := m.Update(common.CommandCompletedMsg{Output: "", Err: errors.New("boom")})

	assert.Nil(t, cmd)
	if assert.Len(t, m.messages, 1) {
		assert.EqualError(t, m.messages[0].error, "boom")
		assert.Equal(t, "", m.messages[0].text)
	}
	assert.Empty(t, m.messageHistory)
}

func TestUpdate_ExpiresMessages(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))

	first := m.add("first", nil)
	m.add("second", nil)

	m.Update(expireMessageMsg{id: first})

	if assert.Len(t, m.messages, 1) {
		assert.Equal(t, "second", m.messages[0].text)
	}
	assert.Empty(t, m.messageHistory)
}

func TestView_StacksFromBottomRight(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))

	m.add("abc", nil)
	m.add("de", nil)

	dl := render.NewDisplayContext()
	m.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 30, 12)))
	views := dl.DrawList()

	if assert.Len(t, views, 2) {
		assert.Contains(t, views[0].Content, "abc")
		assert.Contains(t, views[1].Content, "de")
		assert.GreaterOrEqual(t, views[0].Rect.Min.X, 0)
		assert.GreaterOrEqual(t, views[1].Rect.Min.X, 0)
		assert.Greater(t, views[0].Rect.Min.Y, views[1].Rect.Min.Y)
	}
}

func TestDeleteOldest_RemovesFirstMessage(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))

	m.add("first", nil)
	m.add("second", nil)
	assert.True(t, m.Any())

	m.DeleteOldest()

	if assert.Len(t, m.messages, 1) {
		assert.Equal(t, "second", m.messages[0].text)
	}
}
