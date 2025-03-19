package jj

type GraphRow struct {
	Connections  [][]ConnectionType
	Commit       *Commit
	IsSelected   bool
	IsAffected   bool
	SegmentLines []SegmentedLine
	Indent       int
}

func NewGraphRow() GraphRow {
	return GraphRow{
		Commit:       &Commit{},
		SegmentLines: make([]SegmentedLine, 0),
	}
}

func (r *GraphRow) AddLine(line SegmentedLine) {
	switch len(r.SegmentLines) {
	case 0:
		line.CanHighlight = true
	default:
		if line.ContainsRune('~', r.Indent) {
			line.CanHighlight = false
		} else {
			lastLine := r.SegmentLines[len(r.SegmentLines)-1]
			line.CanHighlight = lastLine.CanHighlight
		}
	}
	r.SegmentLines = append(r.SegmentLines, line)
}
