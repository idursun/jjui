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

func isOperationId(text string) bool {
	if len(text) != 12 {
		return false
	}
	for _, r := range text {
		if !(r >= 'a' && r <= 'f' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func (l *rowLine) FindIdIndex() int {
	for i, segment := range l.Segments {
		if isOperationId(segment.Text) {
			return i
		}
	}
	return -1
}

func newRowLine(segments []*screen.Segment) rowLine {
	return rowLine{Segments: segments}
}
