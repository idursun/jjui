package parser

import (
	"slices"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
)

type RowLineFlags int

const (
	Revision RowLineFlags = 1 << iota
	Highlightable
	Elided
	Description
)

type Row struct {
	Commit     *jj.Commit
	Lines      []*GraphRowLine
	IsAffected bool
	Indent     int
	Previous   *Row
}

func (row *Row) GetSearchableLines() []screen.SearchableLine {
	lines := make([]screen.SearchableLine, len(row.Lines))
	for i, line := range row.Lines {
		lines[i] = line
	}
	return lines
}

func NewGraphRow() Row {
	return Row{
		Commit: &jj.Commit{},
		Lines:  make([]*GraphRowLine, 0),
	}
}

func (row *Row) Get(line int, col int) (rune, bool) {
	if line < 0 || line >= len(row.Lines) {
		return ' ', false
	}
	l := row.Lines[line]
	if col < 0 || col >= len(l.Gutter.Segments) {
		return ' ', false
	}
	g := l.Gutter.Segments[col]
	for _, r := range g.Text {
		return r, true
	}
	return ' ', false
}

func (row *Row) GetLane(line int, col int) uint64 {
	if line < 0 || line >= len(row.Lines) {
		return 0
	}
	if col < 0 || col >= len(row.Lines[line].Gutter.Segments) {
		return 0
	}
	return row.Lines[line].Gutter.Segments[col].Lane
}

func (row *Row) SetLane(line int, col int, lane uint64) {
	if line < 0 || line >= len(row.Lines) {
		return
	}
	if col < 0 || col >= len(row.Lines[line].Gutter.Segments) {
		return
	}
	previousLane := row.Lines[line].Gutter.Segments[col].Lane
	row.Lines[line].Gutter.Segments[col].Lane = previousLane | lane
}

func (row *Row) Extend() GraphGutter {
	type extendResult int
	const (
		No extendResult = iota
		Yes
		Carry
	)
	canExtend := func(text string) extendResult {
		for _, p := range text {
			switch p {
			case '│', '|', '╭', '├', '┐', '┤', '┌', '╮', '┬', '┼', '+', '\\', '.':
				return Yes
			case '╯', '╰', '└', '┴', '┘', ' ', '/':
				return No
			case '─', '-':
				return Carry
			}
		}
		return No
	}

	extendMask := make([]bool, len(row.Lines[0].Gutter.Segments))
	var lastGutter *GraphGutter
	for _, gl := range row.Lines {
		if gl.Flags&Highlightable != Highlightable {
			continue
		}
		for i, s := range gl.Gutter.Segments {
			answer := canExtend(s.Text)
			switch answer {
			case Yes:
				extendMask[i] = true
			case No:
				extendMask[i] = false
			case Carry:
				// Keep previous value.
			}
		}
		lastGutter = &gl.Gutter
	}

	if lastGutter == nil {
		return GraphGutter{Segments: make([]*screen.Segment, 0)}
	}

	if len(extendMask) > len(lastGutter.Segments) {
		extendMask = extendMask[:len(lastGutter.Segments)]
	}
	ret := GraphGutter{
		Segments: make([]*screen.Segment, len(extendMask)),
	}
	for i, b := range extendMask {
		ret.Segments[i] = &screen.Segment{
			Style: lastGutter.Segments[i].Style,
			Text:  " ",
		}
		if b {
			ret.Segments[i].Text = "│"
		}
	}
	return ret
}

func (row *Row) AddLine(line *GraphRowLine) {
	if row.Commit == nil {
		return
	}
	line.chop(row.Indent)
	applyLabels(row.Commit, line.Segments)
	line.Flags = line.labelledFlags()
	if line.Flags == 0 && len(row.Lines) > 0 {
		// Graph-only lines continue the lanes of the revision above them.
		line.Flags = row.Lines[len(row.Lines)-1].Flags & Highlightable
	}
	row.Lines = append(row.Lines, line)
}

func (row *Row) Last(flag RowLineFlags) *GraphRowLine {
	for _, v := range slices.Backward(row.Lines) {
		if v.Flags&flag == flag {
			return v
		}
	}
	return &GraphRowLine{}
}

func (row *Row) RowLinesIter(predicate RowLinesIteratorPredicate) func(yield func(index int, line *GraphRowLine) bool) {
	return func(yield func(index int, line *GraphRowLine) bool) {
		for i := range row.Lines {
			line := row.Lines[i]
			if predicate(line.Flags) {
				if !yield(i, line) {
					return
				}
			}
		}
	}
}

type RowLinesIteratorPredicate func(f RowLineFlags) bool
