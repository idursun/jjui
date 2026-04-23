package details

import (
	"fmt"
)

type status uint8

const (
	Added status = iota
	Deleted
	Modified
	Renamed
	Copied
)

type item struct {
	status   status
	name     string
	fileName string
	selected bool
	conflict bool
}

func (f item) Title() string {
	status := "M"
	switch f.status {
	case Added:
		status = "A"
	case Deleted:
		status = "D"
	case Modified:
		status = "M"
	case Renamed:
		status = "R"
	case Copied:
		status = "C"
	}

	return fmt.Sprintf("%s %s", status, f.name)
}
func (f item) Description() string { return "" }
func (f item) FilterValue() string { return f.name }
