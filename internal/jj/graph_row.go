package jj

type GraphRow struct {
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
		line.Flags = Revision | Highlightable
	default:
		if line.ContainsRune('~', r.Indent) {
			line.Flags = Elided
		} else {
			lastLine := r.SegmentLines[len(r.SegmentLines)-1]
			line.Flags = lastLine.Flags & ^Revision & ^Elided
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
		if line.Flags&Highlightable != 0 {
			lastLine = line
		}
	}
	return lastLine
}

type SegmentedLineIteratorPredicate func(f SegmentedLineFlag) bool

func Including(flags SegmentedLineFlag) SegmentedLineIteratorPredicate {
	return func(f SegmentedLineFlag) bool {
		return f&flags == flags
	}
}

func Excluding(flags SegmentedLineFlag) SegmentedLineIteratorPredicate {
	return func(f SegmentedLineFlag) bool {
		return f&flags != flags
	}
}

func (r *GraphRow) SegmentLinesIter(predicate SegmentedLineIteratorPredicate) func(yield func(line *SegmentedLine) bool) {
	return func(yield func(line *SegmentedLine) bool) {
		for i := range r.SegmentLines {
			line := &r.SegmentLines[i]
			if predicate(line.Flags) {
				if !yield(line) {
					return
				}
			}
		}
	}
}
