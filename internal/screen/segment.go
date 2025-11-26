package screen

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Segment struct {
	Text  string
	Style lipgloss.Style
	Lane  uint64
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
						Text:  text,
						Style: rawSegment.Style,
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
