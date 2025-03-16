package jj

import (
	"bufio"
	"github.com/idursun/jjui/internal/screen"
	"io"
	"strings"
)

type NoTemplateParser struct {
	reader         *bufio.Reader
	segments       []screen.Segment
	segmentIndex   int
	lineEndIndex   int
	lineStartIndex int
	current        screen.Segment
}

func NewNoTemplateParser(reader io.Reader) *NoTemplateParser {
	return &NoTemplateParser{
		reader:       bufio.NewReader(reader),
		segmentIndex: -1,
	}
}

func (p *NoTemplateParser) Parse() []GraphRow {
	var rows []GraphRow
	bytesData, _ := io.ReadAll(p.reader)
	rawSegments := screen.Parse(bytesData)
	// break segments from new lines
	for _, rawSegment := range rawSegments {
		for {
			idx := strings.IndexByte(rawSegment.Text, '\n')
			if idx == -1 {
				break
			}
			text := rawSegment.Text[:idx+1]
			if text != "" {
				p.segments = append(p.segments, screen.Segment{
					Text:   text,
					Params: rawSegment.Params,
				})
			}
			rawSegment.Text = rawSegment.Text[idx+1:]
		}
		if rawSegment.Text != "" {
			p.segments = append(p.segments, rawSegment)
		}
	}

	for p.advance() {
		row := GraphRow{}
		cur := p.lineStartIndex
		row.Commit = &Commit{}
		if p.advanceToIdRestPair() {
			row.Commit.ChangeIdShort = p.segments[p.segmentIndex-1].Text
			row.Commit.ChangeId = row.Commit.ChangeIdShort + p.segments[p.segmentIndex].Text
			if p.advanceToIdRestPair() {
				row.Commit.CommitIdShort = p.segments[p.segmentIndex-1].Text
				row.Commit.CommitId = row.Commit.CommitIdShort + p.segments[p.segmentIndex].Text
			}
		}
		if p.advanceToIdRestPair() {
			p.segmentIndex = p.lineStartIndex
		}

		row.Segments = p.segments[cur : p.lineEndIndex+1]
		row.Connections = make([][]ConnectionType, 1)
		row.Connections[0] = make([]ConnectionType, 1)
		row.Connections[0][0] = GLYPH
		rows = append(rows, row)
	}

	return rows
}

func (p *NoTemplateParser) advance() bool {
	p.segmentIndex++
	if p.segmentIndex >= len(p.segments) {
		return false
	}
	p.current = p.segments[p.segmentIndex]
	if strings.Contains(p.current.Text, "\n") {
		p.lineEndIndex = p.segmentIndex
		p.lineStartIndex = p.segmentIndex + 1
	}
	return true
}

func (p *NoTemplateParser) advanceToIdRestPair() bool {
	for p.advance() {
		if !strings.Contains(p.current.Text, " ") {
			if p.advance() {
				if !strings.Contains(p.current.Text, " ") {
					return true
				}
			}
		}
	}
	return false
}
