package context

import (
	"bytes"
	"context"
	"errors"
	"github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
	"io"
	"log"
	"os"
	"os/exec"
	"slices"
	"sync"
)

type CommandRunner interface {
	RunCommandImmediate(args []string) ([]byte, error)
	RunCommandStreaming(ctx context.Context, args []string) (*StreamingCommand, error)
	RunCommand(args []string, continuations ...tea.Cmd) tea.Cmd
	RunInteractiveCommand(args []string, continuation tea.Cmd) tea.Cmd
}

type MainCommandRunner struct {
	Location string
}

func (a *MainCommandRunner) RunCommandImmediate(args []string) ([]byte, error) {
	c := exec.Command("jj", args...)
	c.Dir = a.Location
	output, err := c.CombinedOutput()
	return bytes.Trim(output, "\n"), err
}

func (a *MainCommandRunner) RunCommandStreaming(ctx context.Context, args []string) (*StreamingCommand, error) {
	c := exec.CommandContext(ctx, "jj", args...)
	c.Dir = a.Location
	pipe, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}
	errPipe, err := c.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err = c.Start(); err != nil {
		return nil, err
	}
	return &StreamingCommand{
		ReadCloser: pipe,
		ErrPipe:    errPipe,
		cmd:        c,
		ctx:        ctx,
	}, nil
}

func (a *MainCommandRunner) RunCommand(args []string, continuations ...tea.Cmd) tea.Cmd {
	commands := make([]tea.Cmd, 0)
	commands = append(commands,
		func() tea.Msg {
			if !slices.Contains(args, "--color") {
				args = append(args, "--color", "always")
			}
			c := exec.Command("jj", args...)
			c.Dir = a.Location
			output, err := c.CombinedOutput()
			return common.CommandCompletedMsg{
				Output: string(output),
				Err:    err,
			}
		})
	commands = append(commands, continuations...)
	return tea.Batch(
		common.CommandRunning(args),
		tea.Sequence(commands...),
	)
}

func (a *MainCommandRunner) RunInteractiveCommand(args []string, continuation tea.Cmd) tea.Cmd {
	c := exec.Command("jj", args...)
	errBuffer := &bytes.Buffer{}
	c.Stderr = errBuffer
	c.Dir = a.Location
	return tea.Batch(
		common.CommandRunning(args),
		tea.ExecProcess(c, func(err error) tea.Msg {
			if err != nil {
				return common.CommandCompletedMsg{Err: err, Output: errBuffer.String()}
			}
			return tea.Batch(continuation, func() tea.Msg {
				return common.CommandCompletedMsg{Err: nil}
			})()
		}),
	)
}

type StreamingCommand struct {
	io.ReadCloser
	ErrPipe io.ReadCloser
	cmd     *exec.Cmd
	ctx     context.Context
	once    sync.Once
}

func (c *StreamingCommand) Close() error {
	var err error
	c.once.Do(func() {
		log.Println("closing streaming command")
		pipeErr := c.ReadCloser.Close()

		if c.ctx.Err() != nil {
			log.Println("killing process due to context cancellation")
			if killErr := c.cmd.Process.Kill(); killErr != nil {
				err = killErr
				return
			}
		}

		log.Println("waiting for command to finish")
		err = c.cmd.Wait()
		if err != nil && (c.ctx.Err() != nil || errors.Is(err, os.ErrClosed)) {
			err = nil
		}

		if pipeErr != nil && err == nil {
			err = pipeErr
		}
	})
	return err
}
