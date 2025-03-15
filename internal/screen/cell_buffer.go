package screen

import (
	"log"
	"strings"
)

type cellBuffer struct {
	grid [][]Segment
}

func Stacked(view1, view2 string, x, y int) string {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	buf := &cellBuffer{}

	// Parse and apply base view
	buf.applyANSI([]byte(view1), 0, 0)
	buf.applyANSI([]byte(view2), x, y)

	return buf.String()
}

func (b *cellBuffer) applyANSI(input []byte, offsetX, offsetY int) {
	parsed := Parse(input)

	currentLine := offsetY
	currentCol := offsetX
	for _, st := range parsed {
		for _, char := range st.Text {
			if char == '\n' {
				currentLine++
				currentCol = offsetX
				continue
			}

			// Expand buffer as needed
			for currentLine >= len(b.grid) {
				b.grid = append(b.grid, []Segment{})
			}
			for currentCol >= len(b.grid[currentLine]) {
				b.grid[currentLine] = append(b.grid[currentLine], Segment{Text: string(' ')})
			}

			// Overwrite cell
			if currentCol < 0 || currentLine < 0 {
				log.Fatalf("line: %d, col: %d", currentLine, currentCol)
			}
			b.grid[currentLine][currentCol] = Segment{
				Text:   string(char),
				Params: st.Params,
			}
			currentCol++
		}
	}
}

func (b *cellBuffer) String() string {
	var segments [][]*Segment

	for _, line := range b.grid {
		var lineSegments []*Segment
		var lastSegment *Segment
		for _, c := range line {
			if lastSegment == nil || !lastSegment.StyleEqual(c) {
				if lastSegment != nil {
					lineSegments = append(lineSegments, lastSegment)
				}
				lastSegment = &Segment{
					Text:   c.Text,
					Params: c.Params,
				}
			} else {
				lastSegment.Text += c.Text
			}
		}
		lineSegments = append(lineSegments, lastSegment)
		segments = append(segments, lineSegments)
	}

	var sb strings.Builder
	for lineNum, lineStyles := range segments {
		if lineNum > 0 {
			sb.WriteByte('\n')
		}
		for _, style := range lineStyles {
			sb.WriteString(style.String())
		}
	}
	return sb.String()
}
