package parser

import (
	"strconv"
	"strings"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
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

func (gr *GraphRowLine) ParseRowPrefixes() (int, string, string, bool) {
	prefixesIdx := -1
	for i, segment := range gr.Segments {
		if strings.Contains(segment.Text, jj.JJUIPrefix) {
			prefixesIdx = i
			break
		}
	}

	if prefixesIdx == -1 {
		return -1, "", "", false
	}
	prefixParts := strings.Split(gr.Segments[prefixesIdx].Text, jj.JJUIPrefix)
	if len(prefixParts) != 4 {
		return -1, "", "", false
	}
	beforePrefix := prefixParts[0]
	changeID := prefixParts[1]
	commitID := prefixParts[2]
	isDivergentStr := prefixParts[3]

	isDivergent, err := strconv.ParseBool(strings.TrimSpace(isDivergentStr))
	if err != nil {
		isDivergent = false
	}

	// Remove changeID, commitID, and isDivergent prefixes, while keeping
	// everything before the prefixes
	gr.Segments[prefixesIdx] = &screen.Segment{Text: beforePrefix}

	return prefixesIdx + 1, changeID, commitID, isDivergent
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

	// Pad with spaces if indent is not fully consumed
	if indent > 0 && len(gr.Gutter.Segments) > 0 {
		lastSegment := gr.Gutter.Segments[len(gr.Gutter.Segments)-1]
		lastSegment.Text += strings.Repeat(" ", indent)
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
