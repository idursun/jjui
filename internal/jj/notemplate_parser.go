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

type SegmentedLine struct {
	Segments []*screen.Segment
}

func (sl SegmentedLine) Extend(indent int) SegmentedLine {
	ret := SegmentedLine{}
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

func NewNoTemplateParser(reader io.Reader) *NoTemplateParser {
	return &NoTemplateParser{
		reader: bufio.NewReader(reader),
	}
}

func (p *NoTemplateParser) Parse() []GraphRow {
	var rows []GraphRow
	bytesData, _ := io.ReadAll(p.reader)
	rawSegments := screen.Parse(bytesData)
	// group segments into lines by breaking segments at new lines
	var segmentedLines []SegmentedLine
	i := 0
	for i < len(rawSegments) {
		currentLine := SegmentedLine{}
		for i < len(rawSegments) {
			rawSegment := &rawSegments[i]
			idx := strings.IndexByte(rawSegment.Text, '\n')
			if idx == -1 {
				currentLine.Segments = append(currentLine.Segments, rawSegment)
				i++
			} else {
				text := rawSegment.Text[:idx]
				currentLine.Segments = append(currentLine.Segments, &screen.Segment{
					Text:   text,
					Params: rawSegment.Params,
				})
				rawSegment.Text = rawSegment.Text[idx+1:]
				break
			}
		}
		segmentedLines = append(segmentedLines, currentLine)
	}

	i = 0
	for i < len(segmentedLines) {
		row := GraphRow{}
		row.Commit = &Commit{}
		segmentedLine := segmentedLines[i]
		changeIdIdx := segmentedLine.getPair(0)
		segmentedLine.Indent = 0
		for j := 0; j < changeIdIdx; j++ {
			segmentedLine.Indent += utf8.RuneCountInString(segmentedLine.Segments[j].Text)
		}
		if changeIdIdx != -1 {
			row.Commit.ChangeIdShort = segmentedLine.Segments[changeIdIdx].Text
			row.Commit.ChangeId = row.Commit.ChangeIdShort + segmentedLine.Segments[changeIdIdx+1].Text
			commitIdIdx := segmentedLine.getPair(changeIdIdx + 2)
			if commitIdIdx != -1 {
				row.Commit.CommitIdShort = segmentedLine.Segments[commitIdIdx].Text
				row.Commit.CommitId = row.Commit.CommitIdShort + segmentedLine.Segments[commitIdIdx+1].Text
			} else {
				log.Fatalln("commit id not found")
			}
		}
		row.SegmentLines = append(row.SegmentLines, segmentedLine)
		i++
		indent := segmentedLine.Indent
		for i < len(segmentedLines) {
			segmentedLine = segmentedLines[i]
			changeIdIdx := segmentedLine.getPair(0)
			if changeIdIdx == -1 {
				segmentedLine.Indent = indent
				row.SegmentLines = append(row.SegmentLines, segmentedLine)
				i++
				continue
			}
			break
		}
		rows = append(rows, row)
	}

	return rows
}
