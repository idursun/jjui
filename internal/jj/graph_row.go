package jj

type GraphRow struct {
	Connections  [][]ConnectionType
	Commit       *Commit
	IsSelected   bool
	IsAffected   bool
	SegmentLines []SegmentedLine
	Indent       int
}
