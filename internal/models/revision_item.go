package models

var _ IItem = (*RevisionItem)(nil)

type RevisionItem struct {
	*Checkable
	Row
	IsAffected bool
}

func (r *RevisionItem) Equals(other IItem) bool {
	if r == nil {
		return false
	}
	otherRevision, ok := other.(*RevisionItem)
	if !ok || otherRevision == nil {
		return false
	}
	return r.Row.Commit.CommitId == otherRevision.Row.Commit.CommitId && r.Row.Commit.ChangeId == otherRevision.Row.Commit.ChangeId
}

func NewRevisionItem(row Row) *RevisionItem {
	return &RevisionItem{
		&Checkable{Checked: false},
		row,
		false,
	}
}
