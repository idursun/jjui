package context

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingRunner struct {
	noopRunner
	runCommandArgs            [][]string
	runCommandWithInputArgs   [][]string
	runInteractiveCommandArgs [][]string
}

func (r *recordingRunner) RunCommand(args []string, _ ...tea.Cmd) tea.Cmd {
	r.runCommandArgs = append(r.runCommandArgs, args)
	return func() tea.Msg { return nil }
}

func (r *recordingRunner) RunCommandWithInput(args []string, _ string, _ ...tea.Cmd) tea.Cmd {
	r.runCommandWithInputArgs = append(r.runCommandWithInputArgs, args)
	return func() tea.Msg { return nil }
}

func (r *recordingRunner) RunInteractiveCommand(args []string, _ tea.Cmd) tea.Cmd {
	r.runInteractiveCommandArgs = append(r.runInteractiveCommandArgs, args)
	return func() tea.Msg { return nil }
}

var errImmutable = errors.New(`Error: Commit abc123 is immutable
Hint: Could not modify commit: abc123
Hint: Immutable commits are used to protect shared history.`)

func TestBuildRetry_NilCases(t *testing.T) {
	cases := []struct {
		name string
		args []string
		err  error
	}{
		{"error is not immutable", []string{"edit", "-r", "abc"}, errors.New("Error: No such revision")},
		{"error is nil", []string{"edit", "-r", "abc"}, nil},
		{"already ignoring immutable", []string{"edit", "-r", "abc", "--ignore-immutable"}, errImmutable},
		{"args use a -- separator", []string{"squash", "-r", "abc", "--", "path"}, errImmutable},
	}

	for _, tc := range cases {
		t.Run("BuildBufferedRetry/"+tc.name, func(t *testing.T) {
			retry := BuildBufferedRetry(&recordingRunner{}, tc.args, nil, nil, tc.err)
			assert.Nil(t, retry)
		})
		t.Run("BuildInteractiveRetry/"+tc.name, func(t *testing.T) {
			retry := BuildInteractiveRetry(&recordingRunner{}, tc.args, nil, tc.err)
			assert.Nil(t, retry)
		})
	}
}

func TestBuildBufferedRetry_UsesRunCommandWhenInputIsNil(t *testing.T) {
	runner := &recordingRunner{}
	retry := BuildBufferedRetry(runner, []string{"edit", "-r", "abc"}, nil, nil, errImmutable)
	require.NotNil(t, retry)

	retry()

	require.Len(t, runner.runCommandArgs, 1)
	assert.Equal(t, []string{"edit", "-r", "abc", "--ignore-immutable"}, runner.runCommandArgs[0])
	assert.Empty(t, runner.runCommandWithInputArgs)
}

func TestBuildBufferedRetry_UsesRunCommandWithInputWhenInputIsSet(t *testing.T) {
	runner := &recordingRunner{}
	input := "new description"
	retry := BuildBufferedRetry(runner, []string{"describe", "-r", "abc", "--stdin"}, &input, nil, errImmutable)
	require.NotNil(t, retry)

	retry()

	require.Len(t, runner.runCommandWithInputArgs, 1)
	assert.Equal(t, []string{"describe", "-r", "abc", "--stdin", "--ignore-immutable"}, runner.runCommandWithInputArgs[0])
	assert.Empty(t, runner.runCommandArgs)
}

func TestBuildInteractiveRetry_AppendsFlagAndReissues(t *testing.T) {
	runner := &recordingRunner{}
	retry := BuildInteractiveRetry(runner, []string{"split", "-r", "abc"}, nil, errImmutable)
	require.NotNil(t, retry)

	retry()

	require.Len(t, runner.runInteractiveCommandArgs, 1)
	assert.Equal(t, []string{"split", "-r", "abc", "--ignore-immutable"}, runner.runInteractiveCommandArgs[0])
}
