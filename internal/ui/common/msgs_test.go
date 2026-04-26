package common

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestQuit_ResetsMode2031BeforeQuit(t *testing.T) {
	cmd := Quit()
	msg := cmd()

	// tea.Sequence returns an unexported sequenceMsg type ([]tea.Cmd).
	// Use reflection to extract the underlying slice.
	val := reflect.ValueOf(msg)
	cmdType := reflect.TypeFor[tea.Cmd]()
	if !assert.Equal(t, reflect.Slice, val.Kind()) ||
		!assert.True(t, val.Type().Elem().AssignableTo(cmdType)) {
		return
	}
	cmds := make([]tea.Cmd, val.Len())
	for i := range cmds {
		cmds[i] = val.Index(i).Interface().(tea.Cmd)
	}

	assert.Len(t, cmds, 2)

	first := cmds[0]()
	raw, ok := first.(tea.RawMsg)
	assert.True(t, ok, "first cmd should produce tea.RawMsg")
	assert.Equal(t, ansi.ResetModeLightDark, raw.Msg)

	second := cmds[1]()
	_, ok = second.(tea.QuitMsg)
	assert.True(t, ok, "second cmd should produce tea.QuitMsg")
}
