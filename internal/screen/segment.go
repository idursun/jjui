package screen

import (
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
)

type Segment struct {
	Text  string
	Style lipgloss.Style
	Lane  uint64
	// Labels holds the jj template labels attached to the text when the
	// output was produced with `--color=debug`.
	Labels []string
}

func (s Segment) HasLabel(label string) bool {
	return slices.Contains(s.Labels, label)
}

// SearchableLine represents a line that contains searchable segments.
type SearchableLine interface {
	GetSegments() []*Segment
}

// Searchable represents a row that can be searched.
type Searchable interface {
	GetSearchableLines() []SearchableLine
}

func (s Segment) String() string {
	return s.Style.Render(s.Text)
}

// BreakNewLinesIter group segments into lines by breaking segments at new lines
func BreakNewLinesIter(rawSegments <-chan *Segment) <-chan []*Segment {
	output := make(chan []*Segment)
	go func() {
		defer close(output)
		currentLine := make([]*Segment, 0)
		for rawSegment := range rawSegments {
			idx := strings.IndexByte(rawSegment.Text, '\n')
			for idx != -1 {
				text := rawSegment.Text[:idx]
				if len(text) > 0 {
					currentLine = append(currentLine, &Segment{
						Text:   text,
						Style:  rawSegment.Style,
						Labels: rawSegment.Labels,
					})
				}
				output <- currentLine
				currentLine = make([]*Segment, 0)
				rawSegment.Text = rawSegment.Text[idx+1:]
				idx = strings.IndexByte(rawSegment.Text, '\n')
			}
			if len(rawSegment.Text) > 0 {
				currentLine = append(currentLine, rawSegment)
			}
		}
		if len(currentLine) > 0 {
			output <- currentLine
		}
	}()
	return output
}
