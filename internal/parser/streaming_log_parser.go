package parser

import (
	"io"
	"unicode/utf8"

	"github.com/idursun/jjui/internal/screen"
)

type ControlMsg int

const (
	RequestMore ControlMsg = iota
	Close
)

type RowBatch struct {
	Rows    []Row
	HasMore bool
}

func ParseRowsStreaming(reader io.Reader, controlChannel <-chan ControlMsg, batchSize int, done <-chan struct{}) <-chan RowBatch {
	rowsChan := make(chan RowBatch, 1)
	go func() {
		defer close(rowsChan)
		var rows []Row
		var row Row
		rawSegments := screen.ParseFromReader(reader)
		for segmentedLine := range screen.BreakNewLinesIter(rawSegments) {
			rowLine := NewGraphRowLine(segmentedLine)
			changeIDIdx, commit := rowLine.ParseRowPrefixes()
			if changeIDIdx != -1 && changeIDIdx != len(rowLine.Segments)-1 {
				previousRow := row
				if len(rows) > batchSize {
					msg, ok := waitForControl(controlChannel, done)
					if !ok {
						return
					}
					switch msg {
					case Close:
						return
					case RequestMore:
						rowsChan <- RowBatch{Rows: rows, HasMore: true}
						rows = nil
					}
				}
				row = NewGraphRow()
				if previousRow.Commit != nil {
					rows = append(rows, previousRow)
					row.Previous = &previousRow
				}
				for j := range changeIDIdx {
					row.Indent += utf8.RuneCountInString(rowLine.Segments[j].Text)
				}
				row.Commit = commit
			}
			row.AddLine(&rowLine)
		}
		if row.Commit != nil {
			rows = append(rows, row)
		}
		if len(rows) > 0 {
			msg, ok := waitForControl(controlChannel, done)
			if !ok {
				return
			}
			switch msg {
			case Close:
				return
			case RequestMore:
				rowsChan <- RowBatch{Rows: rows, HasMore: false}
			}
			return
		}

		if _, ok := waitForControl(controlChannel, done); !ok {
			return
		}
	}()
	return rowsChan
}

func waitForControl(controlChannel <-chan ControlMsg, done <-chan struct{}) (ControlMsg, bool) {
	select {
	case <-done:
		return 0, false
	case msg, ok := <-controlChannel:
		return msg, ok
	}
}
