package jj

import (
	"bufio"
	"github.com/idursun/jjui/internal/screen"
	"io"
	"log"
	"strings"
	"unicode/utf8"
)

type NoTemplateParser struct {
	reader *bufio.Reader
}

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

func (sl SegmentedLine) getPair(start int) int {
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

func NewNoTemplateParser(reader io.Reader) *NoTemplateParser {
	return &NoTemplateParser{
		reader: bufio.NewReader(reader),
	}
}

func (p *NoTemplateParser) Parse() []GraphRow {
	bytesData, _ := io.ReadAll(p.reader)
	rawSegments := screen.Parse(bytesData)

	var rows []GraphRow
	var row GraphRow
	for segmentedLine := range breakNewLinesIter(rawSegments) {
		if changeIdIdx := segmentedLine.getPair(0); changeIdIdx != -1 {
			segmentedLine.Flags = Revision | Highlightable
			previousRow := row
			row = NewGraphRow()
			if previousRow.Commit != nil {
				rows = append(rows, previousRow)
				row.Previous = &previousRow
			}
			for j := 0; j < changeIdIdx; j++ {
				row.Indent += utf8.RuneCountInString(segmentedLine.Segments[j].Text)
			}
			segmentedLine.ChangeIdIdx = changeIdIdx
			row.Commit.ChangeIdShort = segmentedLine.Segments[changeIdIdx].Text
			row.Commit.ChangeId = row.Commit.ChangeIdShort + segmentedLine.Segments[changeIdIdx+1].Text
			commitIdIdx := segmentedLine.getPair(changeIdIdx + 2)
			if commitIdIdx != -1 {
				segmentedLine.CommitIdIdx = commitIdIdx
				row.Commit.CommitIdShort = segmentedLine.Segments[commitIdIdx].Text
				row.Commit.CommitId = row.Commit.CommitIdShort + segmentedLine.Segments[commitIdIdx+1].Text
			} else {
				log.Fatalln("commit id not found")
			}
		}
		row.AddLine(segmentedLine)
	}
	return rows
}

// group segments into lines by breaking segments at new lines
func breakNewLinesIter(rawSegments []screen.Segment) func(yield func(line SegmentedLine) bool) {
	return func(yield func(line SegmentedLine) bool) {
		i := 0
		currentLine := NewSegmentedLine()
		for i < len(rawSegments) {
			rawSegment := &rawSegments[i]
			if idx := strings.IndexByte(rawSegment.Text, '\n'); idx != -1 {
				text := rawSegment.Text[:idx]
				currentLine.Segments = append(currentLine.Segments, &screen.Segment{
					Text:   text,
					Params: rawSegment.Params,
				})
				if !yield(currentLine) {
					return
				}
				currentLine = NewSegmentedLine()
				rawSegment.Text = rawSegment.Text[idx+1:]
			} else {
				currentLine.Segments = append(currentLine.Segments, rawSegment)
				i++
			}
		}
	}
}
