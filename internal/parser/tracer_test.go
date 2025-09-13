package parser

import (
	"bufio"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	models2 "github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common/list"
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
	row1 := models2.NewGraphRow()
	row1.Lines = append(row1.Lines,
		&models2.GraphRowLine{
			Gutter: models2.GraphGutter{
				Segments: []*screen.Segment{
					{Text: "│", Style: lipgloss.NewStyle()},
					{Text: " ", Style: lipgloss.NewStyle()},
					{Text: "◆", Style: lipgloss.NewStyle()},
				},
			},
			Flags: models2.Revision | models2.Highlightable,
		},
		&models2.GraphRowLine{
			Gutter: models2.GraphGutter{
				Segments: []*screen.Segment{
					{Text: "│", Style: lipgloss.NewStyle()},
					{Text: " ", Style: lipgloss.NewStyle()},
					{Text: "│", Style: lipgloss.NewStyle()},
				},
			},
		},
		&models2.GraphRowLine{
			Gutter: models2.GraphGutter{
				Segments: []*screen.Segment{
					{Text: "│", Style: lipgloss.NewStyle()},
					{Text: " ", Style: lipgloss.NewStyle()},
					{Text: "~", Style: lipgloss.NewStyle()},
				},
			},
		},
		&models2.GraphRowLine{
			Gutter: models2.GraphGutter{
				Segments: []*screen.Segment{
					{Text: "├", Style: lipgloss.NewStyle()},
					{Text: "─", Style: lipgloss.NewStyle()},
					{Text: "╯", Style: lipgloss.NewStyle()},
				},
			},
		})

	var rows []*models2.RevisionItem
	rows = append(rows, models2.NewRevisionItem(row1))
	l := list.NewList[*models2.RevisionItem]()
	l.SetItems(rows)
	tracer := NewTracer(l, 0, 0, len(rows))

	assert.True(t, tracer.IsGutterInLane(0, 0, 2))
	assert.True(t, tracer.IsGutterInLane(0, 3, 0))
}

func createLaneMap(rows []models2.Row, cursor, start int) string {
	l := list.NewList[*models2.RevisionItem]()
	var items []*models2.RevisionItem
	for _, row := range rows {
		items = append(items, models2.NewRevisionItem(row))
	}
	l.SetItems(items)
	_ = NewTracer(l, cursor, start, len(rows))

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

func newTestTraceableRow(lines []string) models2.Row {
	row := models2.Row{
		Lines: make([]*models2.GraphRowLine, 0),
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
		gutter := models2.GraphGutter{
			Segments: segments,
		}
		flags := models2.RowLineFlags(0)
		if i == 0 {
			flags |= models2.Revision
		}
		row.Lines = append(row.Lines, &models2.GraphRowLine{Gutter: gutter, Flags: flags})
	}
	return row
}

func createRows(g string) []models2.Row {
	g = strings.TrimSpace(g)
	scanner := bufio.NewScanner(strings.NewReader(g))
	var ret []models2.Row
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
