package context

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

func TestBuildBufferedRetry_DoesNotMutateOriginalArgs(t *testing.T) {
	original := []string{"edit", "-r", "abc"}
	snapshot := append([]string(nil), original...)

	retry := BuildBufferedRetry(&noopRunner{}, original, nil, nil, errors.New("Error: Commit abc is immutable"))
	require.NotNil(t, retry)
	retry()

	require.Equal(t, snapshot, original, "building a retry must not mutate the caller's args slice")
}

type noopRunner struct{}

func (noopRunner) RunCommandImmediate([]string) ([]byte, error) { return nil, nil }
func (noopRunner) RunCommandImmediateWithEnv([]string, []string) ([]byte, error) {
	return nil, nil
}
func (noopRunner) RunCommandStreaming(context.Context, []string) (*StreamingCommand, error) {
	return nil, nil
}
func (noopRunner) RunCommand([]string, ...tea.Cmd) tea.Cmd { return func() tea.Msg { return nil } }
func (noopRunner) RunCommandWithInput([]string, string, ...tea.Cmd) tea.Cmd {
	return func() tea.Msg { return nil }
}
func (noopRunner) RunInteractiveCommand([]string, tea.Cmd) tea.Cmd {
	return func() tea.Msg { return nil }
}
