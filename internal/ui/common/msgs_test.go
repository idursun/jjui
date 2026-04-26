package common

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

// extractSequence pulls the underlying []tea.Cmd out of a tea.Sequence result.
// tea.Sequence wraps its commands in an unexported sequenceMsg type, so we
// have to use reflection to inspect the contents.
func extractSequence(t *testing.T, msg tea.Msg) []tea.Cmd {
	t.Helper()
	val := reflect.ValueOf(msg)
	cmdType := reflect.TypeFor[tea.Cmd]()
	if !assert.Equal(t, reflect.Slice, val.Kind()) ||
		!assert.True(t, val.Type().Elem().AssignableTo(cmdType)) {
		return nil
	}
	cmds := make([]tea.Cmd, val.Len())
	for i := range cmds {
		cmds[i] = val.Index(i).Interface().(tea.Cmd)
	}
	return cmds
}

func TestQuit_ResetsMode2031BeforeQuit(t *testing.T) {
	cmds := extractSequence(t, Quit()())
	assert.Len(t, cmds, 2)

	raw, ok := cmds[0]().(tea.RawMsg)
	assert.True(t, ok, "first cmd should produce tea.RawMsg")
	assert.Equal(t, ansi.ResetModeLightDark, raw.Msg)

	_, ok = cmds[1]().(tea.QuitMsg)
	assert.True(t, ok, "second cmd should produce tea.QuitMsg")
}

func TestSuspend_ResetsMode2031BeforeSuspend(t *testing.T) {
	cmds := extractSequence(t, Suspend()())
	assert.Len(t, cmds, 2)

	raw, ok := cmds[0]().(tea.RawMsg)
	assert.True(t, ok, "first cmd should produce tea.RawMsg")
	assert.Equal(t, ansi.ResetModeLightDark, raw.Msg)

	_, ok = cmds[1]().(tea.SuspendMsg)
	assert.True(t, ok, "second cmd should produce tea.SuspendMsg")
}
