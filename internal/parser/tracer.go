package parser

type LaneTracer interface {
	IsInSameLane(current int) bool
	IsGutterInLane(current int, lineIndex int, segmentIndex int) bool
	UpdateGutterText(current int, lineIndex int, segmentIndex int, text string) string
}

var _ LaneTracer = (*NoopTracer)(nil)

type NoopTracer struct{}

func NewNoopTracer() LaneTracer {
	return NoopTracer{}
}

func (n NoopTracer) IsInSameLane(int) bool {
	return true
}
func (n NoopTracer) IsGutterInLane(int, int, int) bool {
	return true
}
func (n NoopTracer) UpdateGutterText(_ int, _ int, _ int, text string) string {
	return text
}

type Tracer struct {
	rows                 []Row
	nextLaneId           uint64
	highlightedRowLane   uint64
	highlightedLowestBit uint64
}

func NewTracer(rows []Row, cursor int, start int, end int) LaneTracer {
	t := &Tracer{
		rows:       rows,
		nextLaneId: 0,
	}
	//log.Println("Tracing lanes from", start, "to", end)
	t.traceLanes(start, end)

	if cursor >= 0 && cursor < len(t.rows) {
		t.highlightedRowLane = t.getRowLane(cursor)
		t.highlightedLowestBit = t.highlightedRowLane & -t.highlightedRowLane
	}

	return t
}

func (t *Tracer) IsInSameLane(current int) bool {
	currentRowLane := t.getRowLane(current)
	return currentRowLane&t.highlightedLowestBit > 0
}

func (t *Tracer) IsGutterInLane(current int, lineIndex int, segmentIndex int) bool {
	gutterLane := t.getLane(current, lineIndex, segmentIndex)
	return gutterLane&t.highlightedLowestBit > 0
}

func (t *Tracer) UpdateGutterText(current int, lineIndex int, i int, text string) string {
	gutterInLane := t.IsGutterInLane(current, lineIndex, i)
	if gutterInLane {
		rightLane := t.getLane(current, lineIndex, i+1)&t.highlightedLowestBit > 0
		upperLane := t.getLane(current, lineIndex-1, i)&t.highlightedLowestBit > 0
		if text == "├" {
			if rightLane && !upperLane {
				text = "╭"
			} else if !rightLane && upperLane {
				text = "│"
			}
		}
		leftLane := t.getLane(current, lineIndex, i-1)&t.highlightedLowestBit > 0
		if text == "┼" && !rightLane && !leftLane && upperLane {
			text = "│"
		}
		if text == "╭" && upperLane && !rightLane {
			text = "│"
		}
		if text == "─" && upperLane && !rightLane && !leftLane {
			text = "│"
		}
		if text == "┬" && upperLane && !rightLane && !leftLane {
			text = "│"
		}
		if text == "╮" && upperLane && leftLane {
			text = "┤"
		}
	}
	return text
}

func (t *Tracer) traceLanes(start int, end int) {
	if start < 0 || end > len(t.rows) {
		return
	}
	for i := start; i < end; i++ {
		row := t.rows[i]
		for _, line := range row.Lines {
			for _, segment := range line.Gutter.Segments {
				segment.Lane = 0
			}
		}
	}

	t.nextLaneId = 1
	for rowIndex := start; rowIndex < end; rowIndex++ {
		row := t.rows[rowIndex]
		for line := 0; line < len(row.Lines); line++ {
			rowLine := row.Lines[line]
			for index, segment := range rowLine.Gutter.Segments {
				if segment.Text == " " {
					continue
				}
				lane := row.GetLane(line, index)
				if lane == 0 {
					t.traceLane(rowIndex, end, line, index)
					t.nextLaneId = t.nextLaneId << 1
				}
			}
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

func (t *Tracer) traceLane(rowIndex int, endIndex int, lineIndex int, index int) {
	if rowIndex >= endIndex {
		return
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
	currentRow.SetLane(lineIndex, index, t.nextLaneId)

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
			if rowIndex >= endIndex {
				continue
			}
			currentRow = t.rows[rowIndex]
			r = 0
			ch, exists = currentRow.Get(r, c)
			if !exists {
				continue
			}
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
		case ' ': // empty space, continue
			continue
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
