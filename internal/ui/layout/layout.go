package layout

import "github.com/charmbracelet/x/cellbuf"

// Implements a simple layout function mostly copied from charmbracelet/uv and to be replaced by it later

type Constraint interface {
	Apply(size int) int
}

type Percent int

func (p Percent) Apply(size int) int {
	if p < 0 {
		return 0
	}
	if p > 100 {
		return size
	}
	return size * int(p) / 100
}

type Fixed int

func (f Fixed) Apply(size int) int {
	if f < 0 {
		return 0
	}
	if int(f) > size {
		return size
	}
	return int(f)
}

func SplitVertical(area cellbuf.Rectangle, constraint Constraint) (top cellbuf.Rectangle, bottom cellbuf.Rectangle) {
	height := min(constraint.Apply(area.Dy()), area.Dy())
	top = cellbuf.Rectangle{
		Min: area.Min, Max: cellbuf.Pos(area.Max.X, area.Min.Y+height),
	}
	bottom = cellbuf.Rectangle{
		Min: cellbuf.Pos(area.Min.X, area.Min.Y+height), Max: area.Max,
	}
	return
}

func SplitHorizontal(area cellbuf.Rectangle, constraint Constraint) (left cellbuf.Rectangle, right cellbuf.Rectangle) {
	width := min(constraint.Apply(area.Dx()), area.Dx())
	left = cellbuf.Rectangle{
		Min: area.Min, Max: cellbuf.Pos(area.Min.X+width, area.Max.Y),
	}
	right = cellbuf.Rectangle{
		Min: cellbuf.Pos(area.Min.X+width, area.Min.Y), Max: area.Max,
	}
	return
}
