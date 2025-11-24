package oplog

import (
	"github.com/idursun/jjui/internal/screen"
)

type row struct {
	OperationId string
	Lines       []*rowLine
}

type rowLine struct {
	Segments []*screen.Segment
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
