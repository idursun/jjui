package context

import (
	"os/exec"
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/askpass"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type continuationMsg struct{}

// collectMsgs runs cmd and every command nested in the batch or sequence it
// returns, in order.
func collectMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice {
		var msgs []tea.Msg
		for i := range v.Len() {
			if c, ok := v.Index(i).Interface().(tea.Cmd); ok {
				msgs = append(msgs, collectMsgs(c)...)
			}
		}
		return msgs
	}
	return []tea.Msg{msg}
}

func TestRunCommand_ContinuationsRunOnlyOnSuccess(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not found")
	}
	runner := &MainCommandRunner{Location: t.TempDir(), Askpass: &askpass.Server{}}
	continuation := func() tea.Msg { return continuationMsg{} }

	t.Run("success", func(t *testing.T) {
		msgs := collectMsgs(runner.RunCommand([]string{"version"}, continuation))
		require.Len(t, msgs, 3)
		assert.IsType(t, common.CommandRunningMsg{}, msgs[0])
		completed, ok := msgs[1].(common.CommandCompletedMsg)
		require.True(t, ok)
		assert.NoError(t, completed.Err)
		assert.IsType(t, continuationMsg{}, msgs[2])
	})

	t.Run("failure", func(t *testing.T) {
		// the temp dir is not a jj repo, so log fails
		msgs := collectMsgs(runner.RunCommand([]string{"log"}, continuation))
		require.Len(t, msgs, 2)
		completed, ok := msgs[1].(common.CommandCompletedMsg)
		require.True(t, ok)
		assert.Error(t, completed.Err)
	})
}
