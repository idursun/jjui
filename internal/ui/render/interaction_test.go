package render

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/stretchr/testify/assert"
)

type testScrollMsg struct {
	Delta      int
	Horizontal bool
}

func TestProcessMouseEvent_WheelCallsMsgFn(t *testing.T) {
	dl := NewDisplayContext()
	dl.AddInteractionFn(layout.Rect(0, 0, 10, 5), func(msg tea.MouseMsg) tea.Msg {
		delta, horizontal := WheelDelta(msg)
		return testScrollMsg{Delta: delta, Horizontal: horizontal}
	}, InteractionScroll, 0)

	tests := []struct {
		button tea.MouseButton
		want   testScrollMsg
	}{
		{tea.MouseWheelUp, testScrollMsg{Delta: -3}},
		{tea.MouseWheelDown, testScrollMsg{Delta: 3}},
		{tea.MouseWheelLeft, testScrollMsg{Delta: -3, Horizontal: true}},
		{tea.MouseWheelRight, testScrollMsg{Delta: 3, Horizontal: true}},
	}
	for _, tt := range tests {
		msg, handled := dl.ProcessMouseEvent(tea.MouseWheelMsg{X: 2, Y: 2, Button: tt.button})
		assert.True(t, handled)
		assert.Equal(t, tt.want, msg)
	}

	_, handled := dl.ProcessMouseEvent(tea.MouseWheelMsg{X: 20, Y: 2, Button: tea.MouseWheelDown})
	assert.False(t, handled, "wheel outside the region is not handled")
}
