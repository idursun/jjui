package context

import (
	"context"
	"errors"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
)

type AppContext interface {
	Location() string
	KeyMap() config.KeyMappings[key.Binding]
	SelectedItem() SelectedItem
	SetSelectedItem(item SelectedItem) tea.Cmd
	RunCommandImmediate(args []string) ([]byte, error)
	RunCommandStreaming(ctx context.Context, args []string) (*StreamingCommand, error)
	RunCommand(args []string, continuations ...tea.Cmd) tea.Cmd
	RunInteractiveCommand(args []string, continuation tea.Cmd) tea.Cmd
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
