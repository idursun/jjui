package parser

import (
	"io"
	"strings"
	"unicode/utf8"

	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/screen"
)

type ControlMsg int

const (
	RequestMore ControlMsg = iota
	Close
)

func ParseRowsStreaming(reader io.Reader, controlChannel <-chan ControlMsg, batchSize int) (<-chan models.RowBatch, error) {
	rowsChan := make(chan models.RowBatch, 1)
	go func() {
		defer close(rowsChan)
		var rows []models.Row
		var row models.Row
		rawSegments := screen.ParseFromReader(reader)
		for segmentedLine := range screen.BreakNewLinesIter(rawSegments) {
			rowLine := models.NewGraphRowLine(segmentedLine)
			if changeIdIdx := rowLine.FindPossibleChangeIdIdx(); changeIdIdx != -1 && changeIdIdx != len(rowLine.Segments)-1 {
				rowLine.Flags = models.Revision | models.Highlightable
				previousRow := row
				if len(rows) > batchSize {
					select {
					case msg := <-controlChannel:
						switch msg {
						case Close:
							return
						case RequestMore:
							var items []*models.RevisionItem
							for _, r := range rows {
								items = append(items, models.NewRevisionItem(r))
							}
							rowsChan <- models.RowBatch{Items: items, HasMore: true}
							rows = nil
							break
						}
					}
				}
				row = models.NewGraphRow()
				if previousRow.Commit != nil {
					rows = append(rows, previousRow)
					row.Previous = &previousRow
				}
				for j := 0; j < changeIdIdx; j++ {
					row.Indent += utf8.RuneCountInString(rowLine.Segments[j].Text)
				}
				row.Commit.ChangeId = rowLine.Segments[changeIdIdx].Text
				for nextIdx := changeIdIdx + 1; nextIdx < len(rowLine.Segments); nextIdx++ {
					nextSegment := rowLine.Segments[nextIdx]
					if strings.TrimSpace(nextSegment.Text) == "" || strings.ContainsAny(nextSegment.Text, "\n\t\r ") {
						break
					}
					row.Commit.ChangeId += nextSegment.Text
				}
				if commitIdIdx := rowLine.FindPossibleCommitIdIdx(changeIdIdx); commitIdIdx != -1 {
					row.Commit.CommitId = rowLine.Segments[commitIdIdx].Text
				}
			}
			row.AddLine(&rowLine)
		}
		if row.Commit != nil {
			rows = append(rows, row)
		}
		if len(rows) > 0 {
			select {
			case msg := <-controlChannel:
				switch msg {
				case Close:
					return
				case RequestMore:
					var items []*models.RevisionItem
					for _, r := range rows {
						items = append(items, models.NewRevisionItem(r))
					}
					rowsChan <- models.RowBatch{Items: items, HasMore: false}
					rows = nil
					break
				}
			}
		}
		_ = <-controlChannel
	}()
	return rowsChan, nil
}
