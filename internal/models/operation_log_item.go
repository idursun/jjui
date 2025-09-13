package models

var _ IItem = (*OperationLogItem)(nil)

type OperationLogItem struct {
	OperationLogRow
	OperationId string
}

func (o OperationLogItem) Equals(other IItem) bool {
	otherLog, ok := other.(*OperationLogItem)
	if !ok || otherLog == nil {
		return false
	}
	return o.OperationId == otherLog.OperationId
}
