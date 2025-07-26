package parser

type Traceable interface {
	Get(line int, col int) (rune, bool)
	GetNodeIndex() int
}

type TraceableRow struct {
	row *Row
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
				if r == '@' || r == '○' || r == '◆' {
					return j
				}
			}
		}
	}
	return 0
}

type TracedLanes []int

type Tracer struct {
	lanes map[int]TracedLanes
}

func NewTracer() *Tracer {
	return &Tracer{}
}

func (t *Tracer) Trace(row Traceable, tracedLanes TracedLanes) (bool, TracedLanes) {
	index := row.GetNodeIndex()
	for i, lane := range tracedLanes {
		if lane == index {
			// remove col from traced lanes if it exists
			tracedLanes = append(tracedLanes[:i], tracedLanes[i+1:]...)
			rowTraceLanes := t.GetTraceLanes(row)
			return true, append(tracedLanes, rowTraceLanes...)
		}
	}
	return false, tracedLanes
}

func (t *Tracer) GetTraceLanes(row Traceable) TracedLanes {
	index := row.GetNodeIndex()

	type dir int
	const (
		down dir = iota
		left
		right
	)

	type direction struct {
		col  int
		line int
		dir  dir
	}

	var directions []direction
	var tracedLanes TracedLanes
	directions = append(directions, direction{col: index, line: 0, dir: down})
	// implement a breadth-first search to find all lanes that are traced
	for len(directions) > 0 {
		current := directions[0]
		directions = directions[1:]
		r := current.line
		c := current.col
		switch current.dir {
		case down:
			r += 1
		case left:
			c -= 1
		case right:
			c += 1
		}

		ch, exists := row.Get(r, c)
		if !exists {
			tracedLanes = append(tracedLanes, current.col)
			continue
		}
		switch ch {
		case '─':
			directions = append(directions, direction{col: c, line: r, dir: current.dir})
		case '│', '~':
			directions = append(directions, direction{col: c, line: r, dir: down})
		case '┤':
			directions = append(directions, direction{col: c, line: r, dir: down})
		case '┬':
			directions = append(directions, direction{col: c, line: r, dir: down})
			if current.dir == left {
				directions = append(directions, direction{col: c, line: r, dir: left})
			}
			if current.dir == right {
				directions = append(directions, direction{col: c, line: r, dir: right})
			}
		case '├':
			directions = append(directions, direction{col: c, line: r, dir: down})
			if current.dir != left {
				directions = append(directions, direction{col: c, line: r, dir: right})
			}
		case '╯', '┘':
			directions = append(directions, direction{col: c, line: r, dir: left})
		case '╰', '└':
			directions = append(directions, direction{col: c, line: r, dir: right})
		case '╮', '┐':
			directions = append(directions, direction{col: c, line: r, dir: down})
		case '╭', '┌':
			directions = append(directions, direction{col: c, line: r, dir: down})
		}
	}
	return tracedLanes
}
