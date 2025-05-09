package graph

import (
	"github.com/idursun/jjui/internal/screen"
	"io"
	"strings"
	"unicode/utf8"
)

const chunkSize = 50

type ControlMsg int

const (
	RequestMore ControlMsg = iota
	Close
)

type RowBatch struct {
	Rows    []Row
	HasMore bool
}

func ParseRowsStreaming(reader io.Reader, controlChannel <-chan ControlMsg) (<-chan RowBatch, error) {
	rowsChan := make(chan RowBatch, 1)
	go func() {
		defer close(rowsChan)
		var rows []Row
		var row Row
		rawSegments := screen.ParseFromReader(reader)
		for segmentedLine := range screen.BreakNewLinesIter(rawSegments) {
			rowLine := NewGraphRowLine(segmentedLine)
			if changeIdIdx := rowLine.FindPossibleChangeIdIdx(); changeIdIdx != -1 {
				rowLine.Flags = Revision | Highlightable
				previousRow := row
				if len(rows) > chunkSize {
					select {
					case msg := <-controlChannel:
						switch msg {
						case Close:
							return
						case RequestMore:
							rowsChan <- RowBatch{Rows: rows, HasMore: true}
							rows = nil
							break
						}
					}
				}
				row = NewGraphRow()
				if previousRow.Commit != nil {
					rows = append(rows, previousRow)
					row.Previous = &previousRow
				}
				for j := 0; j < changeIdIdx; j++ {
					row.Indent += utf8.RuneCountInString(rowLine.Segments[j].Text)
				}
				rowLine.ChangeIdIdx = changeIdIdx
				row.Commit.ChangeId = rowLine.Segments[changeIdIdx].Text
				for nextIdx := changeIdIdx + 1; nextIdx < len(rowLine.Segments); nextIdx++ {
					nextSegment := rowLine.Segments[nextIdx]
					if strings.TrimSpace(nextSegment.Text) == "" || strings.ContainsAny(nextSegment.Text, "\n\t\r ") {
						break
					}
					row.Commit.ChangeId += nextSegment.Text
				}
				if commitIdIdx := rowLine.FindPossibleCommitIdIdx(changeIdIdx); commitIdIdx != -1 {
					rowLine.CommitIdIdx = commitIdIdx
					row.Commit.CommitId = rowLine.Segments[commitIdIdx].Text
				}
			}
			row.AddLine(&rowLine)
		}
		if row.Commit != nil {
			rows = append(rows, row)
		}
		rowsChan <- RowBatch{Rows: rows, HasMore: false}
	}()
	return rowsChan, nil
}

func ParseRows(reader io.Reader) []Row {
	var rows []Row
	controlChan := make(chan ControlMsg)
	defer close(controlChan)
	streamerChannel, err := ParseRowsStreaming(reader, controlChan)
	if err != nil {
		return nil
	}
	for chunk := range streamerChannel {
		rows = append(rows, chunk.Rows...)
		if !chunk.HasMore {
			break
		}
		controlChan <- RequestMore
	}
	return rows
}
