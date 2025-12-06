package graph

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"sync"

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
// Returns:
// - Streamer: If stdout is successfully opened.
// - Error: Returns the stderr output (warnings are also written to stderr).
func NewGraphStreamer(parentCtx context.Context, runner appContext.CommandRunner, revset string, jjTemplate string) (*GraphStreamer, error) {
	ctx, cancel := context.WithCancel(parentCtx)

	command, err := runner.RunCommandStreaming(ctx, jj.Log(revset, config.Current.Limit, jjTemplate))
	if err != nil {
		cancel()
		return nil, err
	}

	var stderrBuf bytes.Buffer
	var stderrMu sync.Mutex

	// We must read stderr in the background because it may not be closed until the command exits. (e.g. warnings)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := command.ErrPipe.Read(buf)
			if n > 0 {
				stderrMu.Lock()
				stderrBuf.Write(buf[:n])
				stderrMu.Unlock()
			}
			if err != nil {
				// EOF or Context Cancelled
				return
			}
		}
	}()

	// Peek at the first byte of stdout. This blocks ONLY until:
	//   a) jj writes at least 1 byte to stdout, which means there's graph data
	//   b) jj closes stdout/exits, which means failure or no data
	stdoutReader := bufio.NewReader(command)
	_, peekErr := stdoutReader.Peek(1)

	// Non-zero exit or empty stdout
	if peekErr != nil {
		// If we can't read stdout, the command likely failed. We wait for it to exit and gather stderr.
		_ = command.Wait()

		stderrMu.Lock()
		fullStderr := stderrBuf.String()
		stderrMu.Unlock()

		cancel()

		if fullStderr == "" {
			return nil, peekErr // Fallback if no stderr msg but pipe closed
		}
		return nil, errors.New(fullStderr)
	}

	// If we are here, Stdout has data. We grab any warnings accumulated in the buffer.
	stderrMu.Lock()
	warningMsg := stderrBuf.String()
	stderrMu.Unlock()

	controlChan := make(chan parser.ControlMsg, 1)
	batchSize := config.Current.Graph.BatchSize
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	rowsChan, err := parser.ParseRowsStreaming(stdoutReader, controlChan, batchSize)
	if err != nil {
		cancel()
		_ = command.Close()
		return nil, err
	}
	if warningMsg != "" {
		err = errors.New(warningMsg)
	}

	return &GraphStreamer{
		command:     command,
		cancel:      cancel,
		controlChan: controlChan,
		rowsChan:    rowsChan,
		batchSize:   batchSize,
	}, err
}

func (g *GraphStreamer) RequestMore() parser.RowBatch {
	if g.controlChan == nil {
		return parser.RowBatch{}
	}
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
