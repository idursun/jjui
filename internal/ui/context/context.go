package context

import (
	"context"
	"errors"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbletea"
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
	cmd    *exec.Cmd
	ctx    context.Context
	once   sync.Once
	Cancel context.CancelFunc
}

func (c *StreamingCommand) Close() error {
	var err error
	log.Println("closing streaming command")
	c.once.Do(func() {
		log.Println("closing streaming command")
		// First close the pipe
		pipeErr := c.ReadCloser.Close()

		// Then kill the process if ctx is canceled or we're explicitly closing
		if c.ctx.Err() != nil {
			// Context was canceled, ensure process is terminated
			log.Println("killing process due to context cancellation")
			if killErr := c.cmd.Process.Kill(); killErr != nil {
				err = killErr
				return
			}
		}

		// Wait for the process to exit
		log.Println("waiting for command to finish")
		err = c.cmd.Wait()
		if err != nil && (c.ctx.Err() != nil || errors.Is(err, os.ErrClosed)) {
			// If context was canceled or pipe was closed, ignore the error
			err = nil
		}

		if pipeErr != nil && err == nil {
			err = pipeErr
		}
	})
	return err
}
