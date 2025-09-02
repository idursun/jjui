package oplog

import (
	"io"

	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common/models"
)

func newRowLine(segments []*screen.Segment) models.OperationLogRowLine {
	return models.OperationLogRowLine{Segments: segments}
}

func parseRows(reader io.Reader) []*models.OperationLogItem {
	var rows []*models.OperationLogItem
	var r models.OperationLogRow
	rawSegments := screen.ParseFromReader(reader)

	for segmentedLine := range screen.BreakNewLinesIter(rawSegments) {
		rowLine := newRowLine(segmentedLine)
		if opIdIdx := rowLine.FindIdIndex(); opIdIdx != -1 {
			if r.OperationId != "" {
				rows = append(rows, &models.OperationLogItem{OperationId: r.OperationId, OperationLogRow: r})
			}
			r = models.OperationLogRow{OperationId: rowLine.Segments[opIdIdx].Text}
		}
		r.Lines = append(r.Lines, &rowLine)
	}
	rows = append(rows, &models.OperationLogItem{OperationId: r.OperationId, OperationLogRow: r})
	return rows
}
