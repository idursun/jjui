package flash

import (
	"errors"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
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
	m.successStyle = lipgloss.NewStyle() // keep output predictable

	cmd := m.Update(common.CommandCompletedMsg{Output: "  success  ", Err: nil})

	assert.NotNil(t, cmd)
	if assert.Len(t, m.messages, 1) {
		assert.Equal(t, "success", m.messages[0].text)
		assert.Nil(t, m.messages[0].error)
	}
}

func TestUpdate_AddsErrorMessageWithoutExpiry(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.errorStyle = lipgloss.NewStyle()

	cmd := m.Update(common.CommandCompletedMsg{Output: "", Err: errors.New("boom")})

	assert.Nil(t, cmd)
	if assert.Len(t, m.messages, 1) {
		assert.EqualError(t, m.messages[0].error, "boom")
		assert.Equal(t, "", m.messages[0].text)
	}
}

func TestUpdate_ExpiresMessages(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.successStyle = lipgloss.NewStyle()

	first := m.add("first", nil)
	m.add("second", nil)

	m.Update(expireMessageMsg{id: first})

	if assert.Len(t, m.messages, 1) {
		assert.Equal(t, "second", m.messages[0].text)
	}
}

func TestView_StacksFromBottomRight(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.successStyle = lipgloss.NewStyle()
	m.errorStyle = lipgloss.NewStyle()

	m.add("abc", nil)
	m.add("de", nil)

	dl := render.NewDisplayContext()
	m.ViewRect(dl, layout.NewBox(cellbuf.Rect(0, 0, 10, 3)))
	views := dl.DrawList()

	if assert.Len(t, views, 2) {
		w0, _ := lipgloss.Size(views[0].Content)
		w1, _ := lipgloss.Size(views[1].Content)
		assert.Equal(t, "abc", views[0].Content)
		assert.Equal(t, "de", views[1].Content)
		assert.Equal(t, 10-w0, views[0].Rect.Min.X)
		assert.Equal(t, 10-w1, views[1].Rect.Min.X)
		assert.Equal(t, 1, views[0].Rect.Min.Y)
		assert.Equal(t, 0, views[1].Rect.Min.Y)
	}
}

func TestDeleteOldest_RemovesFirstMessage(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.successStyle = lipgloss.NewStyle()

	m.add("first", nil)
	m.add("second", nil)
	assert.True(t, m.Any())

	m.DeleteOldest()

	if assert.Len(t, m.messages, 1) {
		assert.Equal(t, "second", m.messages[0].text)
	}
}
