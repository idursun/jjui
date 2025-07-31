package parser

import (
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
	"strings"
	"unicode"
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
			if row.Commit.CommitId == "" {
				commitIdIdx := line.FindPossibleCommitIdIdx(0)
				if commitIdIdx != -1 {
					row.Commit.CommitId = line.Segments[commitIdIdx].Text
					line.Flags = Revision | Highlightable
				}
			}
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

func isChangeIdLike(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func isHexLike(s string) bool {
	for _, r := range s {
		if !unicode.Is(unicode.Hex_Digit, r) {
			return false
		}
	}
	return true
}

func (row *Row) RowLinesIter(predicate RowLinesIteratorPredicate) func(yield func(line *GraphRowLine) bool) {
	return func(yield func(line *GraphRowLine) bool) {
		for i := range row.Lines {
			line := row.Lines[i]
			if predicate(line.Flags) {
				if !yield(line) {
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
