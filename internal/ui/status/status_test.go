package status

import (
	"errors"
	"testing"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/stretchr/testify/assert"
)

func TestStatus_Update_ExecProcessCompletedMsg(t *testing.T) {
	cases := []struct {
		name           string
		msg            common.ExecProcessCompletedMsg
		expectedMode   string
		expectedPrompt string
		expectedInput  string
		shouldFocus    bool
		expectCmd      bool
	}{
		{
			name: "Execution failed, should restore input",
			msg: common.ExecProcessCompletedMsg{
				Err: errors.New("exit status 1"),
				Msg: common.ExecMsg{
					Line: "invalid command",
					Mode: common.ExecShell,
				},
			},
			expectedMode:   "exec sh",
			expectedPrompt: "$ ",
			expectedInput:  "invalid command",
			shouldFocus:    true,
			expectCmd:      true,
		},
		{
			name: "Execution succeeded",
			msg: common.ExecProcessCompletedMsg{
				Err: nil,
			},
			expectCmd: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &context.MainContext{
				Histories: config.NewHistories(),
			}
			m := New(ctx)
			cmd := m.Update(tt.msg)

			if tt.expectCmd {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}

			if tt.shouldFocus {
				assert.True(t, m.IsFocused())
				assert.Equal(t, tt.expectedMode, m.mode)
				assert.Equal(t, tt.expectedPrompt, m.input.Prompt)
				assert.Equal(t, tt.expectedInput, m.input.Value())
			} else {
				assert.False(t, m.IsFocused())
			}
		})
	}
}
