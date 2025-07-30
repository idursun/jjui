package parser

type Traceable interface {
	Get(line int, col int) (rune, bool)
	GetNodeIndex() int
	GetLane(line int, col int) uint64
	SetLane(line int, col int, lane uint64)
}

type TraceableRow struct {
	row   *Row
	lanes [][]uint64
}

func NewTraceableRow(row *Row) *TraceableRow {
	lanes := make([][]uint64, len(row.Lines))
	for i, line := range row.Lines {
		lanes[i] = make([]uint64, len(line.Gutter.Segments))
	}
	return &TraceableRow{
		row:   row,
		lanes: lanes,
	}
}

func (tr *TraceableRow) Get(line int, col int) (rune, bool) {
	if line < 0 || line >= len(tr.row.Lines) {
		return ' ', false
	}
	l := tr.row.Lines[line]
	if col < 0 || col >= len(l.Gutter.Segments) {
		return ' ', false
	}
	g := l.Gutter.Segments[col]
	for _, r := range g.Text {
		return r, true
	}
	return ' ', false
}

func (tr *TraceableRow) GetNodeIndex() int {
	for _, line := range tr.row.Lines {
		if line.Flags&Revision != Revision {
			continue
		}
		for j, g := range line.Gutter.Segments {
			for _, r := range g.Text {
				if r == '@' || r == '○' || r == '◆' || r == '×' {
					return j
				}
			}
		}
	}
	return 0
}

func (tr *TraceableRow) GetLane(line int, col int) uint64 {
	if line < 0 || line >= len(tr.lanes) {
		return 0
	}
	if col < 0 || col >= len(tr.lanes[line]) {
		return 0
	}
	return tr.lanes[line][col]
}

func (tr *TraceableRow) SetLane(line int, col int, lane uint64) {
	if line < 0 || line >= len(tr.lanes) {
		return
	}
	if col < 0 || col >= len(tr.lanes[line]) {
		return
	}
	previousLane := tr.lanes[line][col]
	tr.lanes[line][col] = previousLane | lane
}

type TracedLanes []int

type Tracer struct {
	rows       []Traceable
	nextLaneId uint64
}

func NewTracer(rows []Traceable) *Tracer {
	t := &Tracer{
		rows:       rows,
		nextLaneId: 0,
	}
	t.TraceLanes()
	return t
}

func (t *Tracer) GetLane(rowIndex int, line int, col int) uint64 {
	return t.rows[rowIndex].GetLane(line, col)
}

func (t *Tracer) GetRowLane(rowIndex int) uint64 {
	currentRow := t.rows[rowIndex]
	index := currentRow.GetNodeIndex()
	lane := currentRow.GetLane(0, index)
	return lane
}

func (t *Tracer) TraceLanes() {
	for i := range t.rows {
		row := t.rows[i]
		index := row.GetNodeIndex()
		lane := row.GetLane(0, index)
		if lane == 0 {
			t.traceLane(i)
		}
	}
}

func (t *Tracer) traceLane(rowIndex int) {
	if t.nextLaneId == 0 {
		t.nextLaneId = 1
	} else {
		t.nextLaneId = t.nextLaneId << 1
	}

	type dir int
	const (
		down dir = iota
		left
		right
	)

	type direction struct {
		rowIndex int
		col      int
		line     int
		dir      dir
	}

	currentRow := t.rows[rowIndex]
	index := currentRow.GetNodeIndex()
	currentRow.SetLane(0, index, t.nextLaneId)

	var directions []direction
	directions = append(directions, direction{rowIndex: rowIndex, line: 0, col: index, dir: down})

	for len(directions) > 0 {
		current := directions[0]
		directions = directions[1:]
		r := current.line
		c := current.col
		rowIndex = current.rowIndex
		currentRow = t.rows[rowIndex]
		switch current.dir {
		case down:
			r += 1
		case left:
			c -= 1
		case right:
			c += 1
		}

		ch, exists := currentRow.Get(r, c)
		if !exists {
			rowIndex++
			if rowIndex >= len(t.rows) {
				continue
			}
			currentRow = t.rows[rowIndex]
			r = 0
			ch, _ = currentRow.Get(r, c)
		}
		currentRow.SetLane(r, c, t.nextLaneId)
		switch ch {
		case '─':
			directions = append(directions, direction{rowIndex: rowIndex, col: c, line: r, dir: current.dir})
		case '│', '~', '○', '◆', '×', '*':
			directions = append(directions, direction{rowIndex: rowIndex, col: c, line: r, dir: down})
		case '┤':
			directions = append(directions, direction{rowIndex: rowIndex, col: c, line: r, dir: down})
		case '┬':
			directions = append(directions, direction{rowIndex: rowIndex, col: c, line: r, dir: down})
			directions = append(directions, direction{rowIndex: rowIndex, col: c, line: r, dir: current.dir})
		case '├':
			directions = append(directions, direction{rowIndex: rowIndex, col: c, line: r, dir: down})
			if current.dir == left {
				currentRow.SetLane(r, c, t.nextLaneId)
			}
			if current.dir != left && t.lookAhead(rowIndex, r, c, 1, '╮') {
				directions = append(directions, direction{rowIndex: rowIndex, col: c, line: r, dir: right})
			}
		case '╯', '┘':
			directions = append(directions, direction{rowIndex: rowIndex, col: c, line: r, dir: left})
		case '╰', '└':
			directions = append(directions, direction{rowIndex: rowIndex, col: c, line: r, dir: right})
		case '╮', '┐':
			directions = append(directions, direction{rowIndex: rowIndex, col: c, line: r, dir: down})
		case '╭', '┌':
			directions = append(directions, direction{rowIndex: rowIndex, col: c, line: r, dir: down})
		}
	}
}

func (t *Tracer) lookAhead(rowIndex int, r int, c int, diff int, expected int32) bool {
	row := t.rows[rowIndex]
	i := c
	for {
		i += diff
		ch, exists := row.Get(r, i)
		if !exists {
			return false
		}
		if ch == expected {
			return true
		}
	}
}
