package jj

import (
	"bufio"
	"github.com/idursun/jjui/internal/screen"
	"io"
	"log"
	"strings"
	"unicode/utf8"
)

type LogParser struct {
	reader *bufio.Reader
}

func NewLogParser(reader io.Reader) *LogParser {
	return &LogParser{
		reader: bufio.NewReader(reader),
	}
}

func (p *LogParser) Parse() []GraphRow {
	var rows []GraphRow
	var row GraphRow
	rawSegments := screen.ParseFromReader(p.reader)

	for segmentedLine := range breakNewLinesIter(rawSegments) {
		if changeIdIdx := segmentedLine.findIdIndex(0); changeIdIdx != -1 {
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
			commitIdIdx := segmentedLine.findIdIndex(changeIdIdx + 2)
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
	if row.Commit != nil {
		rows = append(rows, row)
	}
	return rows
}

// group segments into lines by breaking segments at new lines
func breakNewLinesIter(rawSegments <-chan *screen.Segment) <-chan SegmentedLine {
	output := make(chan SegmentedLine)
	go func() {
		defer close(output)
		currentLine := NewSegmentedLine()
		for rawSegment := range rawSegments {
			idx := strings.IndexByte(rawSegment.Text, '\n')
			for idx != -1 {
				text := rawSegment.Text[:idx]
				if len(text) > 0 {
					currentLine.Segments = append(currentLine.Segments, &screen.Segment{
						Text:   text,
						Params: rawSegment.Params,
					})
				}
				output <- currentLine
				currentLine = NewSegmentedLine()
				rawSegment.Text = rawSegment.Text[idx+1:]
				idx = strings.IndexByte(rawSegment.Text, '\n')
			}
			if len(rawSegment.Text) > 0 {
				currentLine.Segments = append(currentLine.Segments, rawSegment)
			}
		}
		if len(currentLine.Segments) > 0 {
			output <- currentLine
		}
	}()
	return output
}
