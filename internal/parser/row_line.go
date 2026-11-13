package parser

import (
	"strings"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
)

type GraphRowLine struct {
	Segments []*screen.Segment
	Gutter   GraphGutter
	Flags    RowLineFlags
}

func (gr *GraphRowLine) GetSegments() []*screen.Segment {
	return gr.Segments
}

func NewGraphRowLine(segments []*screen.Segment) GraphRowLine {
	return GraphRowLine{
		Segments: segments,
		Gutter:   GraphGutter{Segments: make([]*screen.Segment, 0)},
	}
}

func (gr *GraphRowLine) ParseRowPrefixes() (int, *jj.Commit) {
	prefixesIdx := -1
	for i, segment := range gr.Segments {
		if strings.Contains(segment.Text, jj.JJUIPrefix) {
			prefixesIdx = i
			break
		}
	}

	if prefixesIdx == -1 {
		return -1, nil
	}
	prefixParts := strings.Split(gr.Segments[prefixesIdx].Text, jj.JJUIPrefix)
	if len(prefixParts) != 3 && len(prefixParts) != 4 && len(prefixParts) != 5 && len(prefixParts) != 6 && len(prefixParts) != 7 && len(prefixParts) != 8 {
		return -1, nil
	}
	beforePrefix := prefixParts[0]
	commit := &jj.Commit{
		ChangeId: strings.TrimSpace(prefixParts[1]),
		CommitId: strings.TrimSpace(prefixParts[2]),
	}
	if len(prefixParts) == 6 || len(prefixParts) == 7 || len(prefixParts) == 8 {
		commit.HasBookmarks = strings.TrimSpace(prefixParts[3]) == "true"
		commit.HasLocalBookmarks = strings.TrimSpace(prefixParts[4]) == "true"
		commit.HasWorkspace = strings.TrimSpace(prefixParts[5]) == "true"
	}
	switch len(prefixParts) {
	case 4:
		commit.Root = strings.TrimSpace(prefixParts[3]) == "true"
	case 5:
		commit.Root = strings.TrimSpace(prefixParts[3]) == "true"
		commit.IsEmpty = strings.TrimSpace(prefixParts[4]) == "true"
	case 7, 8:
		commit.Root = strings.TrimSpace(prefixParts[6]) == "true"
		if len(prefixParts) == 8 {
			commit.IsEmpty = strings.TrimSpace(prefixParts[7]) == "true"
		}
	}

	// Remove the structured metadata, while keeping everything before the
	// prefixes.
	gr.Segments[prefixesIdx] = &screen.Segment{Text: beforePrefix}

	return prefixesIdx + 1, commit
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
		style := gr.Gutter.Segments[len(gr.Gutter.Segments)-1].Style
		for range indent {
			gr.Gutter.Segments = append(gr.Gutter.Segments, &screen.Segment{
				Text:  " ",
				Style: style,
			})
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
