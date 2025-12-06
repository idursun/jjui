package parser

import (
	"strings"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
)

type RowLineFlags int

const (
	Revision RowLineFlags = 1 << iota
	Highlightable
	Elided
)

type Row struct {
	Commit     *jj.Commit
	Lines      []*GraphRowLine
	IsAffected bool
	Indent     int
	Previous   *Row
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

func (row *Row) GetNodeIndex() int {
	for _, line := range row.Lines {
		if line.Flags&Revision != Revision {
			continue
		}
		for j, g := range line.Gutter.Segments {
			for _, r := range g.Text {
				if r == '@' || r == '○' || r == '◆' || r == '×' {
					return j
				}
			}
		}
	}
	return 0
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
				extendMask[i] = extendMask[i]
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
	switch len(row.Lines) {
	case 0:
		line.Flags = Revision | Highlightable
		row.Commit.IsWorkingCopy = line.containsRune('@')
		for _, segment := range line.Segments {
			if strings.TrimSpace(segment.Text) == "hidden" {
				row.Commit.Hidden = true
			}
		}
	default:
		if line.containsRune('~') {
			line.Flags = Elided
		} else {
			lastLine := row.Lines[len(row.Lines)-1]
			line.Flags = lastLine.Flags & ^Revision & ^Elided
		}
	}
	row.Lines = append(row.Lines, line)
}

func (row *Row) Last(flag RowLineFlags) *GraphRowLine {
	for i := len(row.Lines) - 1; i >= 0; i-- {
		if row.Lines[i].Flags&flag == flag {
			return row.Lines[i]
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

func Including(flags RowLineFlags) RowLinesIteratorPredicate {
	return func(f RowLineFlags) bool {
		return f&flags == flags
	}
}

func Excluding(flags RowLineFlags) RowLinesIteratorPredicate {
	return func(f RowLineFlags) bool {
		return f&flags != flags
	}
}
