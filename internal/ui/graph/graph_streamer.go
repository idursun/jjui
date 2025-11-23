package graph

import (
	"bufio"
	"context"
	"errors"
	"io"
	"time"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	appContext "github.com/idursun/jjui/internal/ui/context"
)

const DefaultBatchSize = 50

type GraphStreamer struct {
	command     *appContext.StreamingCommand
	cancel      context.CancelFunc
	controlChan chan parser.ControlMsg
	rowsChan    <-chan parser.RowBatch
	batchSize   int
}

// NewGraphStreamer runs `jj log` command with given revset and jjTemplate and
// returns a GraphStreamer
func NewGraphStreamer(ctx appContext.CommandRunner, revset string, jjTemplate string) (*GraphStreamer, error) {
	streamerCtx, cancel := context.WithCancel(context.Background())
	var commandError error

	command, err := ctx.RunCommandStreaming(streamerCtx, jj.Log(revset, config.Current.Limit, jjTemplate))
	if err != nil {
		cancel()
		return nil, err
	}

	// Check stderr with timeout
	errCh := make(chan error, 1)
	go func() {
		errReader := bufio.NewReader(command.ErrPipe)
		data, err := errReader.Peek(1)
		if err == nil && len(data) > 0 {
			errorData, _ := io.ReadAll(errReader)
			errCh <- errors.New(string(errorData))
		} else {
			errCh <- nil
		}
	}()

	// Wait for stderr check with timeout
	select {
	case stderrErr := <-errCh:
		commandError = stderrErr
	case <-time.After(100 * time.Millisecond):
		// Timeout, assume no error and continue
	}

	// Set up stdout processing
	controlChan := make(chan parser.ControlMsg, 1)
	reader := bufio.NewReader(command)

	batchSize := config.Current.Graph.BatchSize
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	rowsChan, err := parser.ParseRowsStreaming(reader, controlChan, batchSize)
	if err != nil {
		cancel()
		_ = command.Close()
		return nil, err
	}

	return &GraphStreamer{
		command:     command,
		cancel:      cancel,
		controlChan: controlChan,
		rowsChan:    rowsChan,
		batchSize:   batchSize,
	}, commandError
}

func (g *GraphStreamer) RequestMore() parser.RowBatch {
	g.controlChan <- parser.RequestMore
	return <-g.rowsChan
}

func (g *GraphStreamer) Close() {
	if g == nil {
		return
	}

	if g.controlChan != nil {
		g.controlChan <- parser.Close
		close(g.controlChan)
		g.controlChan = nil
	}

	if g.cancel != nil {
		g.cancel()
		_ = g.command.Close()
		g.cancel = nil
	}

	g.rowsChan = nil
	g.command = nil
}
