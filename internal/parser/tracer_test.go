package parser

import (
	"bufio"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

type TestTraceableRow struct {
	lines []string
}

func (t TestTraceableRow) Get(line int, col int) (rune, bool) {
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

func (t TestTraceableRow) GetNodeIndex() int {
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
│ *
│ │
`)
	tracer := NewTracer()
	parent, next := tracer.Trace(rows[1], TracedLanes{0})
	assert.False(t, parent)
	assert.Equal(t, TracedLanes{0}, next)
}

func TestGetTraceMaskForCurvedPath(t *testing.T) {
	row := TestTraceableRow{lines: []string{
		"│ *",
		"├─╯",
	}}
	tracer := NewTracer()
	lanes := tracer.GetTraceLanes(row)
	assert.Equal(t, TracedLanes{0}, lanes)
}

func TestTraceCurvedPathConnection(t *testing.T) {
	rows := createRows(`
│ *
├─╯
*
│
`)

	tracer := NewTracer()
	lanes := tracer.GetTraceLanes(rows[0])
	parent, next := tracer.Trace(rows[1], lanes)
	assert.True(t, parent)
	assert.Equal(t, TracedLanes{0}, next)
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
	tracer := NewTracer()
	lanes := tracer.GetTraceLanes(rows[0])
	assert.Equal(t, TracedLanes{0, 2, 4}, lanes)
}

func createRows(g string) []Traceable {
	g = strings.TrimSpace(g)
	scanner := bufio.NewScanner(strings.NewReader(g))
	var ret []Traceable
	var row *TestTraceableRow
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "*") {
			if row != nil {
				ret = append(ret, row)
			}
			row = &TestTraceableRow{lines: []string{}}
		}
		if row != nil {
			row.lines = append(row.lines, line)
		}
	}
	if row != nil {
		ret = append(ret, row)
	}
	return ret
}
