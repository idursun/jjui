package models

import "fmt"

type Status uint8

var (
	Added    Status = 0
	Deleted  Status = 1
	Modified Status = 2
	Renamed  Status = 3
)

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
