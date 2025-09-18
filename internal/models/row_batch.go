package models

type RowBatch struct {
	Items   []*RevisionItem
	HasMore bool
}
