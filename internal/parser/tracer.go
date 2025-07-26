package parser

// TracedLanes is a bitset represented as uint32 where each bit position represents a lane index
type TracedLanes uint32

type Traceable interface {
	Get(line int, col int) (rune, bool)
	GetNodeMask() TracedLanes
}

type TraceableRow struct {
	Row *Row
}

func (tr *TraceableRow) Get(line int, col int) (rune, bool) {
	if line < 0 || line >= len(tr.Row.Lines) {
		return ' ', false
	}
	l := tr.Row.Lines[line]
	if col < 0 || col >= len(l.Gutter.Segments) {
		return ' ', false
	}
	g := l.Gutter.Segments[col]
	for _, r := range g.Text {
		return r, true
	}
	return ' ', false
}

func (tr *TraceableRow) getNodeIndex() int {
	for _, line := range tr.Row.Lines {
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

func (tr *TraceableRow) GetNodeMask() TracedLanes {
	index := tr.getNodeIndex()
	if index < 0 || index >= 32 {
		return 0 // Out of range for our uint32 bitset
	}
	var mask TracedLanes
	mask |= 1 << uint32(index)
	return mask
}

type Tracer struct {
	lanes map[int]TracedLanes
}

func NewTracer() *Tracer {
	return &Tracer{}
}

func (t *Tracer) Trace(row Traceable, tracedLanes TracedLanes) (bool, TracedLanes) {
	nodeMask := row.GetNodeMask()
	if nodeMask < 0 || nodeMask >= 32 {
		return false, tracedLanes // Out of range for our uint32 bitset
	}

	parent := nodeMask&tracedLanes == nodeMask
	return parent, tracedLanes
}

func (t *Tracer) IsLinked(current int, targetIndex int, rows []Traceable) bool {
	if current == targetIndex {
		return true
	}
	if current < 0 || current >= len(rows) {
		return false
	}

	startIndex := current
	endIndex := targetIndex
	if startIndex > endIndex {
		startIndex, endIndex = endIndex, startIndex
	}

	startNodeMask := rows[startIndex].GetNodeMask()
	lanes := t.GetTraceMask(rows[startIndex], startNodeMask)
	isParent := false
	for i := startIndex + 1; i <= endIndex; i++ {
		nodeMask := rows[i].GetNodeMask()
		isParent = nodeMask&lanes == nodeMask
		lanes = t.GetTraceMask(rows[i], lanes)
	}
	return isParent
}

func (t *Tracer) GetTraceMask(row Traceable, mask TracedLanes) TracedLanes {
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
	m := mask
	index := 0
	for m > 0 {
		if m&1 == 1 {
			directions = append(directions, direction{col: index, line: 0, dir: down})
		}
		m >>= 1
		index++
	}
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
			// Only add to tracedLanes if the column is within the valid range for uint32
			if current.col >= 0 && current.col < 32 {
				tracedLanes |= 1 << uint32(current.col)
			}
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
