package models

import "fmt"

type Status uint8

var (
	Added    Status = 0
	Deleted  Status = 1
	Modified Status = 2
	Renamed  Status = 3
)

var _ IItem = (*RevisionFileItem)(nil)

type RevisionFileItem struct {
	*Checkable
	Status   Status
	Name     string
	FileName string
	Conflict bool
}

func (r *RevisionFileItem) Title() string {
	status := "M"
	switch r.Status {
	case Added:
		status = "A"
	case Deleted:
		status = "D"
	case Modified:
		status = "M"
	case Renamed:
		status = "R"
	}

	return fmt.Sprintf("%s %s", status, r.Name)
}

func (r *RevisionFileItem) Equals(other IItem) bool {
	if r == nil {
		return false
	}
	otherFile, ok := other.(*RevisionFileItem)
	if !ok || otherFile == nil {
		return false
	}
	return r.FileName == otherFile.FileName && r.Name == otherFile.Name && r.Status == otherFile.Status && r.Conflict == otherFile.Conflict
}
