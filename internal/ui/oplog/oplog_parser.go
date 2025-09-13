package oplog

import (
	"io"

	models2 "github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/screen"
)

func newRowLine(segments []*screen.Segment) models2.OperationLogRowLine {
	return models2.OperationLogRowLine{Segments: segments}
}

func parseRows(reader io.Reader) []*models2.OperationLogItem {
	var rows []*models2.OperationLogItem
	var r models2.OperationLogRow
	rawSegments := screen.ParseFromReader(reader)

	for segmentedLine := range screen.BreakNewLinesIter(rawSegments) {
		rowLine := newRowLine(segmentedLine)
		if opIdIdx := rowLine.FindIdIndex(); opIdIdx != -1 {
			if r.OperationId != "" {
				rows = append(rows, &models2.OperationLogItem{OperationId: r.OperationId, OperationLogRow: r})
			}
			r = models2.OperationLogRow{OperationId: rowLine.Segments[opIdIdx].Text}
		}
		r.Lines = append(r.Lines, &rowLine)
	}
	rows = append(rows, &models2.OperationLogItem{OperationId: r.OperationId, OperationLogRow: r})
	return rows
}
