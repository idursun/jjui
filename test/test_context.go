package test

import (
	"bytes"
	"github.com/charmbracelet/bubbles/key"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/context"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/stretchr/testify/assert"
)

type ExpectedCommand struct {
	args   []string
	output []byte
	called bool
}

func (e *ExpectedCommand) SetOutput(output []byte) *ExpectedCommand {
	e.output = output
	return e
}

type TestContext struct {
	*testing.T
	selectedItem context.SelectedItem
	expectations map[string][]*ExpectedCommand
}

func (t *TestContext) Location() string {
	return "test"
}

func (t *TestContext) KeyMap() config.KeyMappings[key.Binding] {
	return config.Convert(config.DefaultKeyMappings)
}

func (t *TestContext) SelectedItem() context.SelectedItem {
	return t.selectedItem
}

func (t *TestContext) SetSelectedItem(item context.SelectedItem) tea.Cmd {
	t.selectedItem = item
	return nil
}

func (t *TestContext) RunCommandImmediate(args []string) ([]byte, error) {
	subCommand := args[0]
	if _, ok := t.expectations[subCommand]; !ok {
		assert.Fail(t, "unexpected command", subCommand)
	}
	expectations := t.expectations[subCommand]
	if len(expectations) == 0 {
		assert.Fail(t, "unexpected command", subCommand)
	}
	for _, e := range expectations {
		if assert.Equal(t.T, e.args, args) {
			e.called = true
			return e.output, nil
		}
	}
	assert.Fail(t, "unexpected command", subCommand)
	return nil, nil
}

func (t *TestContext) RunCommandStreaming(args []string) (io.Reader, error) {
	reader, err := t.RunCommandImmediate(args)
	return bytes.NewBuffer(reader), err
}

func (t *TestContext) RunCommand(args []string, continuations ...tea.Cmd) tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds, func() tea.Msg {
		_, _ = t.RunCommandImmediate(args)
		return common.CommandCompletedMsg{}
	})
	cmds = append(cmds, continuations...)
	return tea.Batch(cmds...)
}

func (t *TestContext) RunInteractiveCommand(args []string, continuation tea.Cmd) tea.Cmd {
	return t.RunCommand(args, continuation)
}

func (t *TestContext) Expect(args []string) *ExpectedCommand {
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

func (t *TestContext) Verify() {
	for subCommand, expectations := range t.expectations {
		for _, e := range expectations {
			if !e.called {
				assert.Fail(t, "expected command not called", subCommand)
			}
		}
	}
}

func NewTestContext(t *testing.T) *TestContext {
	return &TestContext{
		T:            t,
		expectations: make(map[string][]*ExpectedCommand),
	}
}
