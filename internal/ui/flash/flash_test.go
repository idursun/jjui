package flash

import (
	"errors"
	"strings"
	"testing"

	"github.com/idursun/jjui/internal/ui/common"
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
