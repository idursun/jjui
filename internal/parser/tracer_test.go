package parser

import (
	"bufio"
	"github.com/rivo/uniseg"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
)

type TestTraceableRow struct {
	lines []string
	lanes [][]uint64
}

func newTestTraceableRow(lines []string) *TestTraceableRow {
	lanes := make([][]uint64, len(lines))
	for i := range lines {
		w := uniseg.StringWidth(lines[i])
		lanes[i] = make([]uint64, w)
	}
	return &TestTraceableRow{
		lines: lines,
		lanes: lanes,
	}
}

func (t *TestTraceableRow) GetLane(line int, col int) uint64 {
	return t.lanes[line][col]
}

func (t *TestTraceableRow) SetLane(line int, col int, lane uint64) {
	if line < 0 || line >= len(t.lines) {
		return
	}
	t.lanes[line][col] = lane
}

func (t *TestTraceableRow) Get(line int, col int) (rune, bool) {
	if line < 0 || line >= len(t.lines) {
		return ' ', false
	}
	if col < 0 || col >= len(t.lines[line]) {
		return ' ', false
	}
	i := 0
	for _, r := range t.lines[line] {
		if i == col {
			return r, true
		}
		i++
	}
	return ' ', false
}

func (t *TestTraceableRow) GetNodeIndex() int {
	for _, line := range t.lines {
		index := 0
		for _, r := range line {
			if r == '*' {
				return index
			}
			index++
		}
	}
	return -1
}

func TestTraceStraightLine(t *testing.T) {
	rows := createRows(`
*
│
*
│
`)
	laneMap := createLaneMap(rows)
	assert.Equal(t, `
1
1
1
1
`, laneMap)
}

func TestTraceCurvedPathConnection(t *testing.T) {
	rows := createRows(`
│ *
├─╯
*
│
`)
	laneMap := createLaneMap(rows)
	assert.Equal(t, `
  1
111
1
1
`, laneMap)
}

func TestMultiBranchTraceMask(t *testing.T) {
	rows := createRows(`
*
├─┬─╮
│ │ *
│ │ │
│ * │
│ ├───╮
* │ │ │
`)
	laneMap := createLaneMap(rows)
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

func createLaneMap(rows []*TestTraceableRow) string {
	var traceableRows []Traceable
	for _, row := range rows {
		traceableRows = append(traceableRows, row)
	}
	_ = NewTracer(traceableRows)
	var sb strings.Builder
	sb.WriteString("\n")
	for _, row := range rows {
		for _, laneLine := range row.lanes {
			for _, lane := range laneLine {
				if lane == 0 {
					sb.WriteString(" ")
				} else {
					sb.WriteString(strconv.Itoa(int(lane)))
				}
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func createRows(g string) []*TestTraceableRow {
	g = strings.TrimSpace(g)
	scanner := bufio.NewScanner(strings.NewReader(g))
	var ret []*TestTraceableRow
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "*") && len(lines) > 0 {
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
