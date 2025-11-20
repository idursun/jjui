package screen

import (
	"strings"

	"github.com/rivo/uniseg"

	"charm.land/lipgloss/v2"
)

type Segment struct {
	Text  string
	Style lipgloss.Style
	Lane  uint64
}

func (s Segment) String() string {
	return s.Style.Render(s.Text)
}

func (s Segment) StyleEqual(other Segment) bool {
	return s.Style.GetForeground() == other.Style.GetForeground() &&
		s.Style.GetBackground() == other.Style.GetBackground() &&
		s.Style.GetBold() == other.Style.GetBold() &&
		s.Style.GetItalic() == other.Style.GetItalic() &&
		s.Style.GetUnderline() == other.Style.GetUnderline() &&
		s.Style.GetStrikethrough() == other.Style.GetStrikethrough() &&
		s.Style.GetBlink() == other.Style.GetBlink() &&
		s.Style.GetReverse() == other.Style.GetReverse() &&
		s.Style.GetFaint() == other.Style.GetFaint()
}

func (s Segment) FindSubstringRange(substr string) (int, int) {
	if s.Text == "" || substr == "" || len(s.Text) < len(substr) {
		return -1, -1
	}
	gr := uniseg.NewGraphemes(s.Text)
	idx := 0
	for gr.Next() {
		from, _ := gr.Positions()
		if len(s.Text[from:]) >= len(substr) && s.Text[from:from+len(substr)] == substr {
			start := idx
			lenGr := 0
			needleGr := uniseg.NewGraphemes(substr)
			for needleGr.Next() {
				lenGr++
			}
			end := start + lenGr
			return start, end
		}
		idx++
	}
	return -1, -1
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
