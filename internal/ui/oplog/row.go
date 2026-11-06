package oplog

import (
	"github.com/idursun/jjui/internal/screen"
)

type row struct {
	OperationId string
	Lines       []*rowLine
}

func (r *row) GetSearchableLines() []screen.SearchableLine {
	lines := make([]screen.SearchableLine, len(r.Lines))
	for i, line := range r.Lines {
		lines[i] = line
	}
	return lines
}

type rowLine struct {
	Segments []*screen.Segment
}

func (rl *rowLine) GetSegments() []*screen.Segment {
	return rl.Segments
}

func (rl *rowLine) FindIdIndex() int {
	for i, segment := range rl.Segments {
		if segment.HasLabel("op_log") && segment.HasLabel("id") {
			return i
		}
	}
	return -1
}

func newRowLine(segments []*screen.Segment) rowLine {
	return rowLine{Segments: segments}
}
