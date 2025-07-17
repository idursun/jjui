package parser

import (
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
	"strings"
	"unicode"
)

type Row struct {
	Commit     *jj.Commit
	Lines      []*GraphRowLine
	IsAffected bool
	Indent     int
	Previous   *Row
}

type RowLineFlags int

const (
	Revision RowLineFlags = 1 << iota
	Highlightable
	Elided
)

type GraphRowLine struct {
	Segments    []*screen.Segment
	Flags       RowLineFlags
	ChangeIdIdx int
	CommitIdIdx int
}

func NewGraphRowLine(segments []*screen.Segment) GraphRowLine {
	return GraphRowLine{
		Segments:    segments,
		ChangeIdIdx: -1,
		CommitIdIdx: -1,
	}
}

type runeTransformer func(r rune) rune

func (gr *GraphRowLine) transform(indent int, transformer runeTransformer) GraphRowLine {
	ret := NewGraphRowLine(make([]*screen.Segment, 0))
	if len(gr.Segments) == 0 {
		return ret
	}

	for _, s := range gr.Segments {
		extended := screen.Segment{
			Style: s.Style,
		}
		var textBuilder strings.Builder
		for _, p := range s.Text {
			if indent <= 0 {
				break
			}
			textBuilder.WriteRune(transformer(p))
			indent--
		}
		extended.Text = textBuilder.String()
		ret.Segments = append(ret.Segments, &extended)
		if indent <= 0 {
			break
		}
	}

	// Pad with spaces if indent is not fully consumed
	if indent > 0 && len(ret.Segments) > 0 {
		lastSegment := ret.Segments[len(ret.Segments)-1]
		lastSegment.Text += strings.Repeat(" ", indent)
	}

	return ret
}

func (gr *GraphRowLine) Chop(indent int) GraphRowLine {
	return gr.transform(indent, func(r rune) rune { return r })
}

func (gr *GraphRowLine) Extend(indent int) GraphRowLine {
	transformer := func(p rune) rune {
		switch p {
		case '│', '╭', '├', '┐', '┤', '┌', '╮', '┬', '┼': // curved, square style
			return '│'
		case '|': // ascii style
			return '|'
		default:
			return ' '
		}
	}
	return gr.transform(indent, transformer)
}

func (gr *GraphRowLine) ContainsRune(r rune, indent int) bool {
	for _, segment := range gr.Segments {
		text := segment.Text
		if len(segment.Text) > indent {
			text = segment.Text[:indent]
		}
		indent -= len(text)
		if strings.ContainsRune(text, r) {
			return true
		}
	}
	return false
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

func (gr *GraphRowLine) FindPossibleChangeIdIdx() int {
	for i, segment := range gr.Segments {
		if isChangeIdLike(segment.Text) {
			return i
		}
	}
	return -1
}

func (gr *GraphRowLine) FindPossibleCommitIdIdx(after int) int {
	for i := after; i < len(gr.Segments); i++ {
		segment := gr.Segments[i]
		if isHexLike(segment.Text) {
			return i
		}
	}
	return -1
}

func NewGraphRow() Row {
	return Row{
		Commit: &jj.Commit{},
		Lines:  make([]*GraphRowLine, 0),
	}
}

func (row *Row) AddLine(line *GraphRowLine) {
	if row.Commit == nil {
		return
	}
	switch len(row.Lines) {
	case 0:
		line.Flags = Revision | Highlightable
		row.Commit.IsWorkingCopy = line.ContainsRune('@', row.Indent)
		for i := line.ChangeIdIdx; i < line.CommitIdIdx; i++ {
			segment := line.Segments[i]
			if strings.TrimSpace(segment.Text) == "hidden" {
				row.Commit.Hidden = true
			}
		}
	default:
		if line.ContainsRune('~', row.Indent) {
			line.Flags = Elided
		} else {
			if row.Commit.CommitId == "" {
				commitIdIdx := line.FindPossibleCommitIdIdx(0)
				if commitIdIdx != -1 {
					line.CommitIdIdx = commitIdIdx
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
