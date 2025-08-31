package models

type Status uint8

var (
	Added    Status = 0
	Deleted  Status = 1
	Modified Status = 2
	Renamed  Status = 3
)

type RevisionFile struct {
	BaseItem
	Checkable
	Name     string
	FileName string
	Status   Status
	Conflict bool
}
