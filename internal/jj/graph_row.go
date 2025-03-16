package jj

import "github.com/idursun/jjui/internal/screen"

type GraphRow struct {
	Connections [][]ConnectionType
	Segments    []screen.Segment
	Commit      *Commit
	IsSelected  bool
	IsAffected  bool
}
