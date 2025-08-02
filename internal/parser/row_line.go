package parser

import (
	"github.com/idursun/jjui/internal/screen"
	"strings"
)

type GraphRowLine struct {
	Segments []*screen.Segment
	Gutter   GraphGutter
	Flags    RowLineFlags
}

func NewGraphRowLine(segments []*screen.Segment) GraphRowLine {
	return GraphRowLine{
		Segments: segments,
		Gutter:   GraphGutter{Segments: make([]*screen.Segment, 0)},
	}
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

func (gr *GraphRowLine) chop(indent int) {
	if len(gr.Segments) == 0 {
		return
	}
	segments := gr.Segments
	gr.Segments = make([]*screen.Segment, 0)

	for i, s := range segments {
		extended := screen.Segment{
			Style: s.Style,
		}
		var textBuilder strings.Builder
		for _, p := range s.Text {
			if indent <= 0 {
				break
			}
			textBuilder.WriteRune(p)
			indent--
		}
		extended.Text = textBuilder.String()
		gr.Gutter.Segments = append(gr.Gutter.Segments, &extended)
		if len(extended.Text) < len(s.Text) {
			gr.Segments = append(gr.Segments, &screen.Segment{
				Text:  s.Text[len(extended.Text):],
				Style: s.Style,
			})
		}
		if indent <= 0 && len(segments)-i-1 > 0 {
			gr.Segments = segments[i+1:]
			break
		}
	}

	// Pad with spaces if indent is not fully consumed
	if indent > 0 && len(gr.Segments) > 0 {
		lastSegment := gr.Segments[len(gr.Segments)-1]
		lastSegment.Text += strings.Repeat(" ", indent)
	}

	// break gutter into segments per rune
	segments = gr.Gutter.Segments
	gr.Gutter.Segments = make([]*screen.Segment, 0)
	for _, s := range segments {
		for _, p := range s.Text {
			extended := screen.Segment{
				Text:  string(p),
				Style: s.Style,
			}
			gr.Gutter.Segments = append(gr.Gutter.Segments, &extended)
		}
	}
}

func (gr *GraphRowLine) containsRune(r rune) bool {
	for _, segment := range gr.Gutter.Segments {
		if strings.ContainsRune(segment.Text, r) {
			return true
		}
	}
	return false
}
