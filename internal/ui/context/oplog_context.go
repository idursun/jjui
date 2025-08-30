package context

import (
	"bytes"
	"io"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
)

type OplogContext struct {
	context *MainContext
	Rows    []Row
	Cursor  int
}

type Row struct {
	OperationId string
	Lines       []*rowLine
}

type rowLine struct {
	Segments []*screen.Segment
}

func (l *rowLine) FindIdIndex() int {
	for i, segment := range l.Segments {
		if len(segment.Text) == 12 {
			return i
		}
	}
	return -1
}

func newRowLine(segments []*screen.Segment) rowLine {
	return rowLine{Segments: segments}
}

func (o *OplogContext) Load() {
	go func() {
		output, err := o.context.RunCommandImmediate(jj.OpLog(config.Current.OpLog.Limit))
		if err != nil {
			panic(err)
		}

		rows := parseRows(bytes.NewReader(output))
		o.Rows = rows
		o.context.App.Send("")
	}()
}

func (o *OplogContext) GetCurrentOperationId() string {
	if len(o.Rows) == 0 {
		return ""
	}
	return o.Rows[o.Cursor].OperationId
}

func (o *OplogContext) Prev() {
	if o.Cursor > 0 {
		o.Cursor--
	}
}

func (o *OplogContext) Next() {
	if o.Cursor < len(o.Rows)-1 {
		o.Cursor++
	}
}

func parseRows(reader io.Reader) []Row {
	var rows []Row
	var r Row
	rawSegments := screen.ParseFromReader(reader)

	for segmentedLine := range screen.BreakNewLinesIter(rawSegments) {
		rowLine := newRowLine(segmentedLine)
		if opIdIdx := rowLine.FindIdIndex(); opIdIdx != -1 {
			if r.OperationId != "" {
				rows = append(rows, r)
			}
			r = Row{OperationId: rowLine.Segments[opIdIdx].Text}
		}
		r.Lines = append(r.Lines, &rowLine)
	}
	rows = append(rows, r)
	return rows
}
