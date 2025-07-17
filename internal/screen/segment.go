package screen

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Segment struct {
	Text     string
	Style    lipgloss.Style
	Reversed bool
}

func splitString(str, searchString string) []string {
	index := strings.Index(str, searchString)
	if index == -1 {
		return []string{str}
	}

	before := str[:index]
	after := str[index+len(searchString):]

	return []string{before, searchString, after}
}

func (s Segment) Reverse(text string) []*Segment {
	ret := make([]*Segment, 0)
	for _, part := range splitString(s.Text, text) {
		if part == "" {
			continue
		}
		ret = append(ret, &Segment{
			Text:     part,
			Style:    s.Style,
			Reversed: part == text,
		})
	}
	return ret
}

func (s Segment) String() string {
	if s.Text == "\n" {
		return s.Text
	}

	style := s.Style
	if s.Reversed {
		style = style.Reverse(true)
	}

	return style.Render(s.Text)
}

func (s Segment) StyleEqual(other Segment) bool {
	return s.Style.String() == other.Style.String()
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
