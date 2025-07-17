package screen

import (
	"log"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rivo/uniseg"
)

type cellBuffer struct {
	grid [][]gridCell
}

type gridCell struct {
	Segment
	width int
}

var emptyCell = gridCell{
	Segment: Segment{
		Text:  "",
		Style: lipgloss.NewStyle(),
	},
	width: 0,
}
var spaceCell = gridCell{
	Segment: Segment{
		Text:  " ",
		Style: lipgloss.NewStyle(),
	},
	width: 1,
}

func Stacked(view1, view2 string, x, y int) string {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	buf := &cellBuffer{}

	buf.applyANSI([]byte(view1), 0, 0)
	buf.applyANSI([]byte(view2), x, y)

	return buf.String()
}

func (b *cellBuffer) applyANSI(input []byte, offsetX, offsetY int) {
	for len(b.grid) <= offsetY {
		b.grid = append(b.grid, []gridCell{})
	}

	if offsetX == 0 && offsetY == 0 && len(b.grid) == 0 {
		b.grid = [][]gridCell{{}}
		b.merge(input, 0, 0)
	} else {
		b.merge(input, offsetX, offsetY)
	}
}

func (b *cellBuffer) merge(input []byte, offsetX, offsetY int) {
	parsed := Parse(input)

	currentLine := offsetY
	currentCol := offsetX

	for len(b.grid) <= currentLine {
		b.grid = append(b.grid, []gridCell{})
	}

	for _, st := range parsed {
		gr := uniseg.NewGraphemes(st.Text)
		for gr.Next() {
			cluster := gr.Str()
			if cluster == "\n" {
				currentLine++
				currentCol = offsetX

				for len(b.grid) <= currentLine {
					b.grid = append(b.grid, []gridCell{})
				}
				continue
			}

			if currentCol < 0 || currentLine < 0 {
				log.Fatalf("line: %d, col: %d", currentLine, currentCol)
			}

			charWidth := gr.Width()

			for len(b.grid[currentLine]) <= currentCol+charWidth-1 {
				b.grid[currentLine] = append(b.grid[currentLine], spaceCell)
			}

			if currentCol > 0 && currentCol < len(b.grid[currentLine]) && b.grid[currentLine][currentCol].width == 0 {
				b.grid[currentLine][currentCol-1] = spaceCell
			}

			if currentCol+charWidth-1 < len(b.grid[currentLine])-1 &&
				b.grid[currentLine][currentCol+charWidth].width == 0 {
				b.grid[currentLine][currentCol+charWidth] = spaceCell
			}

			c := gridCell{
				Segment: Segment{
					Text:  cluster,
					Style: st.Style,
				},
				width: charWidth,
			}

			b.grid[currentLine][currentCol] = c

			if charWidth == 2 && currentCol+1 < len(b.grid[currentLine]) {
				b.grid[currentLine][currentCol+1] = emptyCell
			}

			currentCol += charWidth
		}
	}
}

func (b *cellBuffer) String() string {
	var segments [][]*Segment

	for _, line := range b.grid {
		var lineSegments []*Segment
		var lastSegment *Segment
		for i := 0; i < len(line); i++ {
			c := &line[i]
			if c.width == 0 {
				continue
			}
			if lastSegment == nil || !lastSegment.StyleEqual(c.Segment) {
				if lastSegment != nil {
					lineSegments = append(lineSegments, lastSegment)
				}
				lastSegment = &Segment{
					Text:  c.Text,
					Style: c.Style,
				}
			} else {
				lastSegment.Text += c.Text
			}
		}
		if lastSegment != nil {
			lineSegments = append(lineSegments, lastSegment)
		}
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
