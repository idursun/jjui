package jj

type GraphRow struct {
	Connections  [][]ConnectionType
	Commit       *Commit
	IsSelected   bool
	IsAffected   bool
	SegmentLines []SegmentedLine
	Indent       int
	Previous     *GraphRow
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

func (r *GraphRow) Last(flag SegmentedLineFlag) *SegmentedLine {
	for i := len(r.SegmentLines) - 1; i >= 0; i-- {
		if r.SegmentLines[i].Flags&flag == flag {
			return &r.SegmentLines[i]
		}
	}
	var lastLine *SegmentedLine
	for i := range r.SegmentLines {
		line := &r.SegmentLines[i]
		if line.CanHighlight {
			lastLine = line
		}
	}
	return lastLine
}

func (r *GraphRow) HighlightableSegmentLines() func(yield func(line *SegmentedLine) bool) {
	return func(yield func(line *SegmentedLine) bool) {
		for i := range r.SegmentLines {
			line := &r.SegmentLines[i]
			if line.CanHighlight {
				if !yield(line) {
					return
				}
			}
		}
	}
}

func (r *GraphRow) RemainingSegmentLines() func(yield func(line *SegmentedLine) bool) {
	return func(yield func(line *SegmentedLine) bool) {
		for i := range r.SegmentLines {
			line := &r.SegmentLines[i]
			if !line.CanHighlight {
				if !yield(line) {
					return
				}
			}
		}
	}
}
