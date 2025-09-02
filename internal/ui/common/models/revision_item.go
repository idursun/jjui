package models

type RevisionItem struct {
	*Checkable
	Row
	IsAffected bool
}

func NewRevisionItem(row Row) *RevisionItem {
	return &RevisionItem{
		&Checkable{Checked: false},
		row,
		false,
	}
}
