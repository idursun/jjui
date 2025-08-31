package context

import (
	"bytes"
	"io"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/context/models"
	"github.com/idursun/jjui/internal/ui/helpers"
)

type OplogContext struct {
	CommandRunner
	UI
	*OpLogList
}

type OpLogList struct {
	*helpers.List[models.OperationLogItem]
}

func NewOpLogContext(commandRunner CommandRunner, ui UI) *OplogContext {
	return &OplogContext{
		commandRunner,
		ui,
		&OpLogList{
			List: helpers.NewList[models.OperationLogItem](),
		},
	}
}

func (o *OplogContext) Load() {
	go func() {
		output, err := o.RunCommandImmediate(jj.OpLog(config.Current.OpLog.Limit))
		if err != nil {
			panic(err)
		}

		rows := parseRows(bytes.NewReader(output))
		o.Items = rows
		o.UI.Update()
	}()
}

func findIdIndex(segments []*screen.Segment) int {
	for i, segment := range segments {
		if len(segment.Text) == 12 {
			return i
		}
	}
	return -1
}

func parseRows(reader io.Reader) []models.OperationLogItem {
	var rows []models.OperationLogItem
	var r models.OperationLogItem
	rawSegments := screen.ParseFromReader(reader)

	for segmentedLine := range screen.BreakNewLinesIter(rawSegments) {
		if opIdIdx := findIdIndex(segmentedLine); opIdIdx != -1 {
			if r.OperationId != "" {
				rows = append(rows, r)
			}
			r = models.OperationLogItem{
				BaseItem:    models.BaseItem{},
				OperationId: segmentedLine[opIdIdx].Text,
				Lines:       nil,
			}
		}
		r.Lines = append(r.Lines, segmentedLine)
	}
	rows = append(rows, r)
	return rows
}
