package oplog

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpLogCloseIntent(t *testing.T) {
	m := &Model{
		context: &context.MainContext{},
		rows: []row{
			{
				OperationId: "op1",
			},
		},
		cursor: 0,
	}

	cmd := m.Update(intents.OpLogClose{})

	require.NotNil(t, cmd)

	msg := cmd()
	msgs := []tea.Msg{}

	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, batchCmd := range batch {
			if batchCmd != nil {
				batchMsg := batchCmd()
				msgs = append(msgs, batchMsg)
			}
		}
	}

	hasClose := false
	hasRefresh := false
	hasSelectionChanged := false
	for _, m := range msgs {
		switch m.(type) {
		case common.CloseViewMsg:
			hasClose = true
		case common.RefreshMsg:
			hasRefresh = true
		case common.SelectionChangedMsg:
			hasSelectionChanged = true
		}
	}

	assert.True(t, hasClose, "expected CloseViewMsg to be sent for OpLogClose intent")
	assert.True(t, hasRefresh, "expected RefreshMsg to be sent for OpLogClose intent")
	assert.True(t, hasSelectionChanged, "expected SelectionChangedMsg to be sent for OpLogClose intent")
}

func TestOpLogNavigateIntent(t *testing.T) {
	m := &Model{
		context: &context.MainContext{},
		rows: []row{
			{OperationId: "op1"},
			{OperationId: "op2"},
			{OperationId: "op3"},
		},
		cursor: 0,
	}
	m.listRenderer = render.NewListRenderer(OpLogScrollMsg{})

	cmd := m.Update(intents.OpLogNavigate{Delta: 1, IsPage: false})
	if cmd != nil {
		cmd()
	}

	assert.Equal(t, 1, m.cursor, "expected cursor to move down by 1")

	cmd = m.Update(intents.OpLogNavigate{Delta: -1, IsPage: false})
	if cmd != nil {
		cmd()
	}

	assert.Equal(t, 0, m.cursor, "expected cursor to move back to 0")
}
