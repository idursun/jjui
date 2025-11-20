package parser

import (
	"bufio"
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/screen"
	"github.com/stretchr/testify/assert"
)

func TestTraceStraightLine(t *testing.T) {
	rows := createRows(`
○
│
○
│
`)
	laneMap := createLaneMap(rows, 0, 0)
	assert.Equal(t, `
1
1
1
1
`, laneMap)
}

func TestTraceCurvedPathConnection(t *testing.T) {
	rows := createRows(`
│ ○
├─╯
○
│
`)
	laneMap := createLaneMap(rows, 0, 0)
	assert.Equal(t, `
1 2
322
3
3
`, laneMap)
}

func TestTraceCurvedPathConnectionScrolled(t *testing.T) {
	rows := createRows(`
○
│  
│ ○
├─╯
○
│
`)
	_ = createLaneMap(rows, 0, 0)
	laneMap := createLaneMap(rows, 1, 0)
	assert.Equal(t, `
1
1
1 2
322
3
3
`, laneMap)
}

func TestMultiBranchTraceMask(t *testing.T) {
	rows := createRows(`
○
├─┬─╮
│ │ ○
│ │ │
│ ○ │
│ ├───╮
○ │ │ │
`)
	laneMap := createLaneMap(rows, 0, 0)
	assert.Equal(t, `
1
11111
1 1 1
1 1 1
1 1 1
1 11111
1 1 1 1
`, laneMap)
}

func TestTracer_IsGutterInLane(t *testing.T) {
	row1 := NewGraphRow()
	row1.Lines = append(row1.Lines,
		&GraphRowLine{
			Gutter: GraphGutter{
				Segments: []*screen.Segment{
					{Text: "│", Style: lipgloss.NewStyle()},
					{Text: " ", Style: lipgloss.NewStyle()},
					{Text: "◆", Style: lipgloss.NewStyle()},
				},
			},
			Flags: Revision | Highlightable,
		},
		&GraphRowLine{
			Gutter: GraphGutter{
				Segments: []*screen.Segment{
					{Text: "│", Style: lipgloss.NewStyle()},
					{Text: " ", Style: lipgloss.NewStyle()},
					{Text: "│", Style: lipgloss.NewStyle()},
				},
			},
		},
		&GraphRowLine{
			Gutter: GraphGutter{
				Segments: []*screen.Segment{
					{Text: "│", Style: lipgloss.NewStyle()},
					{Text: " ", Style: lipgloss.NewStyle()},
					{Text: "~", Style: lipgloss.NewStyle()},
				},
			},
		},
		&GraphRowLine{
			Gutter: GraphGutter{
				Segments: []*screen.Segment{
					{Text: "├", Style: lipgloss.NewStyle()},
					{Text: "─", Style: lipgloss.NewStyle()},
					{Text: "╯", Style: lipgloss.NewStyle()},
				},
			},
		})

	var rows []Row
	rows = append(rows, row1)
	tracer := NewTracer(rows, 0, 0, len(rows))
	assert.True(t, tracer.IsGutterInLane(0, 0, 2))
	assert.True(t, tracer.IsGutterInLane(0, 3, 0))
}

func createLaneMap(rows []Row, cursor, start int) string {
	_ = NewTracer(rows, cursor, start, len(rows))
	var sb strings.Builder
	sb.WriteString("\n")
	for _, row := range rows {
		for _, laneLine := range row.Lines {
			for _, lane := range laneLine.Gutter.Segments {
				if lane.Lane == 0 {
					sb.WriteString(" ")
				} else {
					sb.WriteString(strconv.Itoa(int(lane.Lane)))
				}
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func newTestTraceableRow(lines []string) Row {
	row := Row{
		Lines: make([]*GraphRowLine, 0),
	}
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var segments []*screen.Segment
		for _, r := range line {
			segments = append(segments, &screen.Segment{
				Text:  string(r),
				Style: lipgloss.NewStyle(),
			})
		}
		gutter := GraphGutter{
			Segments: segments,
		}
		flags := RowLineFlags(0)
		if i == 0 {
			flags |= Revision
		}
		row.Lines = append(row.Lines, &GraphRowLine{Gutter: gutter, Flags: flags})
	}
	return row
}

func createRows(g string) []Row {
	g = strings.TrimSpace(g)
	scanner := bufio.NewScanner(strings.NewReader(g))
	var ret []Row
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "○") && len(lines) > 0 {
			row := newTestTraceableRow(lines)
			lines = make([]string, 0)
			ret = append(ret, row)
		}
		lines = append(lines, line)
	}
	if len(lines) > 0 {
		ret = append(ret, newTestTraceableRow(lines))
	}
	return ret
}
