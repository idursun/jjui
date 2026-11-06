package oplog

import (
	"io"

	"github.com/idursun/jjui/internal/screen"
)

func parseRows(reader io.Reader) []row {
	var rows []row
	var r row
	rawSegments := screen.ParseLabelledFromReader(reader)

	for segmentedLine := range screen.BreakNewLinesIter(rawSegments) {
		rowLine := newRowLine(segmentedLine)
		if opIdIdx := rowLine.FindIdIndex(); opIdIdx != -1 {
			if r.OperationId != "" {
				rows = append(rows, r)
			}
			r = row{OperationId: rowLine.Segments[opIdIdx].Text}
		}
		r.Lines = append(r.Lines, &rowLine)
	}
	rows = append(rows, r)
	return rows
}
