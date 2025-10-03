package test

import (
	"bytes"
	"context"
	"io"
	"slices"
	"sync"
	"testing"

	appContext "github.com/idursun/jjui/internal/ui/context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/stretchr/testify/assert"
)

type ExpectedCommand struct {
	args   []string
	output []byte
	called bool
	err    error
}

func (e *ExpectedCommand) SetOutput(output []byte) *ExpectedCommand {
	e.output = output
	return e
}

func (e *ExpectedCommand) SetError(err error) *ExpectedCommand {
	e.err = err
	return e
}

type CommandRunner struct {
	*testing.T
	expectations map[string][]*ExpectedCommand
	mutex        sync.Mutex
}

func (t *CommandRunner) RunCommandImmediate(args []string) ([]byte, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	subCommand := args[0]
	expectations, ok := t.expectations[subCommand]
	if !ok || len(expectations) == 0 {
		assert.Fail(t, "unexpected command", subCommand)
	}

	for _, e := range expectations {
		if slices.Equal(e.args, args) {
			e.called = true
			return e.output, e.err
		}
	}
	assert.Fail(t, "unexpected command", subCommand)
	return nil, nil
}

func (t *CommandRunner) RunCommandStreaming(_ context.Context, args []string) (*appContext.StreamingCommand, error) {
	reader, err := t.RunCommandImmediate(args)
	return &appContext.StreamingCommand{
		ReadCloser: io.NopCloser(bytes.NewReader(reader)),
		ErrPipe:    nil,
	}, err
}

func (t *CommandRunner) RunCommand(args []string, continuations ...tea.Cmd) tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds, func() tea.Msg {
		output, err := t.RunCommandImmediate(args)
		return common.CommandCompletedMsg{Output: string(output), Err: err}
	})
	cmds = append(cmds, continuations...)
	return tea.Batch(cmds...)
}

func (t *CommandRunner) RunInteractiveCommand(args []string, continuation tea.Cmd) tea.Cmd {
	return t.RunCommand(args, continuation)
}

func (t *CommandRunner) Expect(args []string) *ExpectedCommand {
	subCommand := args[0]
	if _, ok := t.expectations[subCommand]; !ok {
		t.expectations[subCommand] = make([]*ExpectedCommand, 0)
	}
	e := &ExpectedCommand{
		args: args,
	}
	t.expectations[subCommand] = append(t.expectations[subCommand], e)
	return e
}

func (t *CommandRunner) Verify() {
	for subCommand, subCommandExpectations := range t.expectations {
		for _, e := range subCommandExpectations {
			if !e.called {
				assert.Fail(t, "expected command not called", subCommand)
			}
		}
	}
}

func (t *CommandRunner) IsVerified() bool {
	for _, subCommandExpectations := range t.expectations {
		for _, e := range subCommandExpectations {
			if !e.called {
				return false
			}
		}
	}
	return true
}

func NewTestCommandRunner(t *testing.T) *CommandRunner {
	return &CommandRunner{
		T:            t,
		expectations: make(map[string][]*ExpectedCommand),
	}
}

func NewTestContext(commandRunner appContext.CommandRunner) *appContext.MainContext {
	return &appContext.MainContext{
		CommandRunner: commandRunner,
		SelectedItem:  nil,
	}
}
