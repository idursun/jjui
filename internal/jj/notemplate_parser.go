package jj

import (
	"bufio"
	"github.com/idursun/jjui/internal/screen"
	"io"
	"strings"
)

type NoTemplateParser struct {
	reader       *bufio.Reader
	segments     []screen.Segment
	segmentIndex int
	current      screen.Segment
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
	for _, rawSegment := range rawSegments {
		for {
			idx := strings.IndexByte(rawSegment.Text, '\n')
			if idx == -1 {
				break
			}
			p.segments = append(p.segments, screen.Segment{
				Text:   rawSegment.Text[:idx+1],
				Params: rawSegment.Params,
			})
			rawSegment.Text = rawSegment.Text[idx+1:]
		}
		p.segments = append(p.segments, rawSegment)
	}
	for p.advance() {
		if strings.ContainsFunc(p.current.Text, isGlyph()) {
			row := GraphRow{
				Commit: &Commit{
					IsWorkingCopy: true,
				},
			}
			p.advance()
			row.Commit.ChangeIdShort = p.current.Text
			p.advance()
			row.Commit.ChangeId = row.Commit.ChangeIdShort + p.current.Text
			p.advance()
			row.Commit.Author = p.current.Text
			p.advance()
			row.Commit.Timestamp = p.current.Text
			for p.advance() && !strings.Contains(p.current.Text, "\n") {
			}
			row.Commit.CommitIdShort = p.segments[p.segmentIndex-2].Text
			row.Commit.CommitId = row.Commit.CommitIdShort + p.segments[p.segmentIndex-1].Text
			rows = append(rows, row)
		}
	}

	return rows
}

func (p *NoTemplateParser) advance() bool {
	p.segmentIndex++
	if p.segmentIndex >= len(p.segments) {
		return false
	}
	p.skipWhiteSpace()
	p.current = p.segments[p.segmentIndex]
	return true
}

func (p *NoTemplateParser) skipWhiteSpace() {
	for p.segmentIndex < len(p.segments) && strings.TrimSpace(p.segments[p.segmentIndex].Text) == "" {
		p.segmentIndex++
	}
}

var glyph = '○'
var conflict = '×'
var workingCopy = '@'
var immutable = '◆'

func isGlyph() func(r rune) bool {
	return func(r rune) bool {
		return r == glyph || r == workingCopy || r == conflict || r == immutable
	}
}
