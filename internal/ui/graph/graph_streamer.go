package graph

import (
	"bufio"
	"context"
	"github.com/idursun/jjui/internal/jj"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"log"
)

const DefaultBatchSize = 50

// GraphStreamer encapsulates the streaming of graph rows
type GraphStreamer struct {
	command     *appContext.StreamingCommand
	cancel      context.CancelFunc
	controlChan chan ControlMsg
	rowsChan    <-chan RowBatch
	tag         uint64
	batchSize   int
}

// NewGraphStreamer creates a new graph streamer
func NewGraphStreamer(ctx appContext.AppContext, revset string, tag uint64) (*GraphStreamer, error) {
	streamerCtx, cancel := context.WithCancel(context.Background())

	command, err := ctx.RunCommandStreaming(streamerCtx, jj.Log(revset))
	if err != nil {
		cancel()
		return nil, err
	}

	command.Cancel = cancel
	controlChan := make(chan ControlMsg, 1)
	rowsChan, _ := ParseRowsStreaming(bufio.NewReader(command), controlChan, DefaultBatchSize)

	return &GraphStreamer{
		command:     command,
		cancel:      cancel,
		controlChan: controlChan,
		rowsChan:    rowsChan,
		tag:         tag,
		batchSize:   DefaultBatchSize,
	}, nil
}

// RequestMore requests more rows from the stream
func (g *GraphStreamer) RequestMore() RowBatch {
	log.Println("requesting more rows from graph streamer")
	g.controlChan <- RequestMore
	return <-g.rowsChan
}

// Close releases all resources
func (g *GraphStreamer) Close() {
	if g == nil {
		return
	}

	if g.controlChan != nil {
		g.controlChan <- Close
		log.Println("closing graph streamer control channel")
		close(g.controlChan)
		g.controlChan = nil
	}

	if g.cancel != nil {
		log.Println("canceling graph streamer context")
		g.cancel()
		_ = g.command.Close()
		g.cancel = nil
	}

	g.rowsChan = nil
	g.command = nil
}

// Tag returns the streamer's tag
func (g *GraphStreamer) Tag() uint64 {
	return g.tag
}
