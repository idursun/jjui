package parser

type Tracer struct {
	rows       []Row
	nextLaneId uint64
}

func NewTracer(rows []Row, start int, end int) *Tracer {
	t := &Tracer{
		rows:       rows,
		nextLaneId: 0,
	}
	t.traceLanes(start, end)
	return t
}

func (t *Tracer) IsInSameLane(current int, cursor int) bool {
	currentRowLane := t.getRowLane(current)
	highlightedRowLane := t.getRowLane(cursor)
	lowestBit := highlightedRowLane & -highlightedRowLane
	inLane := currentRowLane&lowestBit > 0
	return inLane
}

func (t *Tracer) IsGutterInLane(current int, cursor int, lineIndex int, segmentIndex int) bool {
	gutterLane := t.getLane(current, lineIndex, segmentIndex)
	highlightedRowLane := t.getRowLane(cursor)
	lowestBit := highlightedRowLane & -highlightedRowLane
	gutterInLane := gutterLane&lowestBit > 0
	return gutterInLane
}

func (t *Tracer) UpdateGutterText(current int, cursor int, lineIndex int, i int, text string) string {
	gutterInLane := t.IsGutterInLane(current, cursor, lineIndex, i)
	highlightedRowLane := t.getRowLane(cursor)
	lowestBit := highlightedRowLane & -highlightedRowLane
	if gutterInLane && text == "├" {
		rightLane := t.getLane(current, lineIndex, i+1)&lowestBit > 0
		upperLane := t.getLane(current, lineIndex-1, i)&lowestBit > 0

		if rightLane && !upperLane {
			text = "╭"
		} else if !rightLane && upperLane {
			text = "│"
		}
	}
	return text
}

func (t *Tracer) traceLanes(start int, end int) {
	for i := start; i < end; i++ {
		row := t.rows[i]
		for _, line := range row.Lines {
			for _, segment := range line.Gutter.Segments {
				segment.Lane = 0
			}
		}
	}
	for i := start; i < end; i++ {
		row := t.rows[i]
		index := row.GetNodeIndex()
		lane := row.GetLane(0, index)
		if lane == 0 {
			t.traceLane(i)
		}
	}
}

func (t *Tracer) getLane(rowIndex int, line int, col int) uint64 {
	return t.rows[rowIndex].GetLane(line, col)
}

func (t *Tracer) getRowLane(rowIndex int) uint64 {
	currentRow := t.rows[rowIndex]
	index := currentRow.GetNodeIndex()
	lane := currentRow.GetLane(0, index)
	return lane
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
		default:
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
