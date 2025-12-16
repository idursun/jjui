package oplog

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelKeyRefreshesPreview(t *testing.T) {
	m := &Model{
		ViewNode:   common.NewViewNode(0, 0),
		MouseAware: common.NewMouseAware(),
		context:    &context.MainContext{},
		rows: []row{
			{
				OperationId: "op1",
			},
		},
		cursor: 0,
		keymap: config.Current.GetKeyMap(),
	}

	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}

	// Send the Cancel key
	cmd := m.Update(keyMsg)

	require.NotNil(t, cmd)

	msg := cmd()
	msgs := []tea.Msg{}

	// The command should be a Batch with Close, Refresh, and SelectionChanged
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

	assert.True(t, hasClose, "expected hasClose to be sent when Cancel key is pressed")
	assert.True(t, hasRefresh, "expected RefreshMsg to be sent when Cancel key is pressed")
	assert.True(t, hasSelectionChanged, "expected SelectionChangedMsg to be sent when Cancel key is pressed")
}
