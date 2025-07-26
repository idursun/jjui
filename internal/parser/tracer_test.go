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

func (t TestTraceableRow) GetNodeMask() TracedLanes {
	index := t.getNodeIndex()
	if index < 0 {
		return 0
	}
	mask := TracedLanes(1 << index)
	return mask
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

func (t TestTraceableRow) getNodeIndex() int {
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
	parent, next := tracer.Trace(rows[1], rows[0].GetNodeMask())
	assert.False(t, parent)
	assert.Equal(t, TracedLanes(0b1), next)
	assert.False(t, tracer.IsLinked(0, 1, rows))
	assert.False(t, tracer.IsLinked(1, 0, rows))
}

func TestGetTraceMaskForCurvedPath(t *testing.T) {
	row := TestTraceableRow{lines: []string{
		"│ *",
		"├─╯",
	}}
	tracer := NewTracer()
	lanes := tracer.GetTraceMask(row, row.GetNodeMask())
	assert.Equal(t, TracedLanes(0b1), lanes)
}

func TestTraceCurvedPathConnection(t *testing.T) {
	rows := createRows(`
│ *
├─╯
*
│
`)

	tracer := NewTracer()
	lanes := tracer.GetTraceMask(rows[0], rows[0].GetNodeMask())
	parent, next := tracer.Trace(rows[1], lanes)
	assert.True(t, parent)
	assert.Equal(t, TracedLanes(0b1), next)
	assert.True(t, tracer.IsLinked(0, 1, rows))
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
	lanes := tracer.GetTraceMask(rows[0], rows[0].GetNodeMask())
	assert.Equal(t, TracedLanes(0b10101), lanes)

	assert.True(t, tracer.IsLinked(0, 1, rows))
	assert.True(t, tracer.IsLinked(0, 2, rows))
	assert.True(t, tracer.IsLinked(0, 3, rows))
}

func TestComplexMergePathLinks(t *testing.T) {
	rows := createRows(`
│ *
│ ├─╮
* │ │
├─╯ │
*   │
├─╮ │
│ * │
│ │ │
`)
	tracer := NewTracer()
	assert.True(t, tracer.IsLinked(0, 2, rows), "0 should be linked to 2")
	assert.False(t, tracer.IsLinked(0, 1, rows), "0 should not be linked to 1")
	assert.True(t, tracer.IsLinked(1, 2, rows), "1 should be linked to 2")
	assert.True(t, tracer.IsLinked(1, 3, rows), "1 should be linked to 3")
}

func TestDisconnectedMergePathLinks(t *testing.T) {
	rows := createRows(`
│ *
├─╯
*
│
│ *
├─╯
`)
	tracer := NewTracer()
	assert.True(t, tracer.IsLinked(0, 1, rows), "0 should be linked to 1")
	assert.False(t, tracer.IsLinked(0, 2, rows), "0 should not be linked to 2")
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
