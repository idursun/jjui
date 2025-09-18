package models

import (
	"github.com/idursun/jjui/internal/screen"
)

type OperationLogRow struct {
	OperationId string
	Lines       []*OperationLogRowLine
}

type OperationLogRowLine struct {
	Segments []*screen.Segment
}

func (l *OperationLogRowLine) FindIdIndex() int {
	for i, segment := range l.Segments {
		if len(segment.Text) == 12 {
			return i
		}
	}
	return -1
}
