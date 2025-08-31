package models

import "github.com/idursun/jjui/internal/screen"

type OperationLogItem struct {
	BaseItem
	OperationId string
	Lines       [][]*screen.Segment
}
