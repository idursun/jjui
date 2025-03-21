package jj

import (
	"github.com/idursun/jjui/internal/screen"
	"strings"
)

type SegmentedLineFlag int

const (
	Revision SegmentedLineFlag = 1 << iota
	Highlightable
	Elided
)

type SegmentedLine struct {
	Segments    []*screen.Segment
	Flags       SegmentedLineFlag
	ChangeIdIdx int
	CommitIdIdx int
}

func NewSegmentedLine() SegmentedLine {
	return SegmentedLine{
		Segments:    make([]*screen.Segment, 0),
		ChangeIdIdx: -1,
		CommitIdIdx: -1,
	}
}

func (sl SegmentedLine) Extend(indent int) SegmentedLine {
	ret := NewSegmentedLine()
	for _, s := range sl.Segments {
		extended := screen.Segment{
			Params: s.Params,
		}
		text := ""
		for _, p := range s.Text {
			if p == '│' || p == '╭' || p == '├' || p == '┐' || p == '┤' || p == '┌' { // curved, square style
				p = '│'
			} else if p == '|' { //ascii style
				p = '|'
			} else {
				p = ' '
			}
			indent--
			text += string(p)
			if indent <= 0 {
				break
			}
		}
		extended.Text = text
		ret.Segments = append(ret.Segments, &extended)
		if indent <= 0 {
			break
		}
	}
	for indent > 0 {
		ret.Segments[len(ret.Segments)-1].Text += " "
		indent--
	}
	return ret
}

func (sl SegmentedLine) findIdIndex(start int) int {
	for i := start; i < len(sl.Segments); i++ {
		cur := sl.Segments[i].Text
		if cur != "" && !strings.Contains(cur, " ") {
			n := i + 1
			if n < len(sl.Segments) {
				cur = sl.Segments[n].Text
				if cur != "" && !strings.Contains(cur, " ") {
					return i
				}
			}
		}
	}
	return -1
}

func (sl SegmentedLine) ContainsRune(r rune, indent int) bool {
	for _, segment := range sl.Segments {
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
