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

func ParseRowsStreaming(reader io.Reader, controlChannel <-chan ControlMsg, batchSize int) (<-chan RowBatch, error) {
	rowsChan := make(chan RowBatch, 1)
	go func() {
		defer close(rowsChan)
		var rows []Row
		var row Row
		rawSegments := screen.ParseFromReader(reader)
		for segmentedLine := range screen.BreakNewLinesIter(rawSegments) {
			rowLine := NewGraphRowLine(segmentedLine)
			changeIDIdx, changeID, commitID, _ := rowLine.ParseRowPrefixes()
			if changeIDIdx != -1 && changeIDIdx != len(rowLine.Segments)-1 {
				rowLine.Flags = Revision | Highlightable
				previousRow := row
				if len(rows) > batchSize {
					switch <-controlChannel {
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
				row.Commit.ChangeId = changeID
				row.Commit.CommitId = commitID
			}
			row.AddLine(&rowLine)
		}
		if row.Commit != nil {
			rows = append(rows, row)
		}
		if len(rows) > 0 {
			switch <-controlChannel {
			case Close:
				return
			case RequestMore:
				rowsChan <- RowBatch{Rows: rows, HasMore: false}
			}
		}
		<-controlChannel
	}()
	return rowsChan, nil
}
