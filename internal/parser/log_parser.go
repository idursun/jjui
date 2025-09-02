package parser

import (
	"io"

	"github.com/idursun/jjui/internal/ui/common/models"
)

func ParseRows(reader io.Reader) []*models.RevisionItem {
	var rows []*models.RevisionItem
	controlChan := make(chan ControlMsg)
	defer close(controlChan)
	streamerChannel, err := ParseRowsStreaming(reader, controlChan, 50)
	if err != nil {
		return nil
	}
	for {
		controlChan <- RequestMore
		chunk := <-streamerChannel
		rows = append(rows, chunk.Items...)
		if !chunk.HasMore {
			break
		}
	}
	return rows
}
