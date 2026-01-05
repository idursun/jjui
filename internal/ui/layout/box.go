package layout

import "github.com/charmbracelet/x/cellbuf"

// Box wraps a cellbuf.Rectangle to provide a fluent API for layout calculations.
// It enables declarative layout syntax instead of manual rectangle arithmetic.
type Box struct {
	R cellbuf.Rectangle
}

// NewBox creates a new Box wrapping the given rectangle.
func NewBox(r cellbuf.Rectangle) Box {
	return Box{R: r}
}

// Spec represents a layout specification that determines how space should be allocated.
// Both the existing Fixed/Percent types and the new FillSpec implement this interface.
type Spec interface {
	// calc calculates the size for this spec given the total available size.
	// total is the original dimension size
	// remaining is space left after fixed/pct allocations
	// fillWeight is the total weight of all Fill specs
	calc(total int, remaining int, fillWeight float64) int
}

// Fixed represents a fixed size in cells.
type Fixed int

// Percent represents a percentage (0-100) of the available space.
type Percent int

// Fixed implements Spec interface
func (f Fixed) calc(total int, remaining int, fillWeight float64) int {
	size := int(f)
	if size < 0 {
		return 0
	}
	if size > total {
		return total
	}
	return size
}

// Percent implements Spec interface
func (p Percent) calc(total int, remaining int, fillWeight float64) int {
	pct := int(p)
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return total
	}
	return total * pct / 100
}

// FillSpec allocates space proportionally from remaining space after fixed/pct allocations.
// The weight determines the proportion: Fill(2) gets twice the space of Fill(1).
type FillSpec float64

func (f FillSpec) calc(total int, remaining int, fillWeight float64) int {
	if remaining <= 0 || fillWeight <= 0 {
		return 0
	}
	weight := float64(f)
	if weight <= 0 {
		return 0
	}
	// Allocate proportionally based on weight
	size := int(float64(remaining) * weight / fillWeight)
	if size < 0 {
		return 0
	}
	return size
}

// Fill creates a Spec that fills remaining space with the given weight.
// Multiple Fill specs share remaining space proportionally by their weights.
func Fill(weight float64) Spec {
	return FillSpec(weight)
}

// Inset returns a new Box with n cells of padding on all sides.
func (b Box) Inset(n int) Box {
	return Box{R: b.R.Inset(n)}
}

// V splits the box vertically (top to bottom) according to the given specs.
// Returns a slice of boxes, one for each spec.
func (b Box) V(specs ...Spec) []Box {
	if len(specs) == 0 {
		return []Box{b}
	}

	height := b.R.Dy()
	if height <= 0 {
		// Return zero-height boxes
		result := make([]Box, len(specs))
		for i := range result {
			result[i] = Box{R: cellbuf.Rectangle{Min: b.R.Min, Max: b.R.Min}}
		}
		return result
	}

	// First pass: calculate sizes and track what's consumed
	sizes := make([]int, len(specs))
	consumed := 0
	fillWeight := 0.0

	// Calculate fill weight and fixed/pct sizes
	for i, spec := range specs {
		switch s := spec.(type) {
		case FillSpec:
			fillWeight += float64(s)
		case Fixed:
			size := s.calc(height, 0, 0)
			sizes[i] = size
			consumed += size
		case Percent:
			size := s.calc(height, 0, 0)
			sizes[i] = size
			consumed += size
		}
	}

	// Calculate remaining space for Fill specs
	remaining := height - consumed
	if remaining < 0 {
		remaining = 0
	}

	// Second pass: calculate Fill sizes
	fillAllocated := 0
	for i, spec := range specs {
		if _, ok := spec.(FillSpec); ok {
			sizes[i] = spec.calc(height, remaining, fillWeight)
			fillAllocated += sizes[i]
		}
	}

	// Handle rounding errors: give remainder to last Fill spec
	if fillWeight > 0 && remaining > fillAllocated {
		remainder := remaining - fillAllocated
		// Find last Fill spec and add remainder
		for i := len(specs) - 1; i >= 0; i-- {
			if _, ok := specs[i].(FillSpec); ok {
				sizes[i] += remainder
				break
			}
		}
	}

	// Create boxes from sizes
	result := make([]Box, len(specs))
	y := b.R.Min.Y
	for i, size := range sizes {
		if y >= b.R.Max.Y {
			// No space left, return zero-size boxes
			result[i] = Box{R: cellbuf.Rectangle{
				Min: cellbuf.Pos(b.R.Min.X, b.R.Max.Y),
				Max: cellbuf.Pos(b.R.Max.X, b.R.Max.Y),
			}}
		} else {
			nextY := y + size
			if nextY > b.R.Max.Y {
				nextY = b.R.Max.Y
			}
			result[i] = Box{R: cellbuf.Rectangle{
				Min: cellbuf.Pos(b.R.Min.X, y),
				Max: cellbuf.Pos(b.R.Max.X, nextY),
			}}
			y = nextY
		}
	}

	return result
}

// H splits the box horizontally (left to right) according to the given specs.
// Returns a slice of boxes, one for each spec.
func (b Box) H(specs ...Spec) []Box {
	if len(specs) == 0 {
		return []Box{b}
	}

	width := b.R.Dx()
	if width <= 0 {
		// Return zero-width boxes
		result := make([]Box, len(specs))
		for i := range result {
			result[i] = Box{R: cellbuf.Rectangle{Min: b.R.Min, Max: b.R.Min}}
		}
		return result
	}

	// First pass: calculate sizes and track what's consumed
	sizes := make([]int, len(specs))
	consumed := 0
	fillWeight := 0.0

	// Calculate fill weight and fixed/pct sizes
	for i, spec := range specs {
		switch s := spec.(type) {
		case FillSpec:
			fillWeight += float64(s)
		case Fixed:
			size := s.calc(width, 0, 0)
			sizes[i] = size
			consumed += size
		case Percent:
			size := s.calc(width, 0, 0)
			sizes[i] = size
			consumed += size
		}
	}

	// Calculate remaining space for Fill specs
	remaining := width - consumed
	if remaining < 0 {
		remaining = 0
	}

	// Second pass: calculate Fill sizes
	fillAllocated := 0
	for i, spec := range specs {
		if _, ok := spec.(FillSpec); ok {
			sizes[i] = spec.calc(width, remaining, fillWeight)
			fillAllocated += sizes[i]
		}
	}

	// Handle rounding errors: give remainder to last Fill spec
	if fillWeight > 0 && remaining > fillAllocated {
		remainder := remaining - fillAllocated
		// Find last Fill spec and add remainder
		for i := len(specs) - 1; i >= 0; i-- {
			if _, ok := specs[i].(FillSpec); ok {
				sizes[i] += remainder
				break
			}
		}
	}

	// Create boxes from sizes
	result := make([]Box, len(specs))
	x := b.R.Min.X
	for i, size := range sizes {
		if x >= b.R.Max.X {
			// No space left, return zero-size boxes
			result[i] = Box{R: cellbuf.Rectangle{
				Min: cellbuf.Pos(b.R.Max.X, b.R.Min.Y),
				Max: cellbuf.Pos(b.R.Max.X, b.R.Max.Y),
			}}
		} else {
			nextX := x + size
			if nextX > b.R.Max.X {
				nextX = b.R.Max.X
			}
			result[i] = Box{R: cellbuf.Rectangle{
				Min: cellbuf.Pos(x, b.R.Min.Y),
				Max: cellbuf.Pos(nextX, b.R.Max.Y),
			}}
			x = nextX
		}
	}

	return result
}

// CutTop cuts h cells from the top, returning the top box and the rest.
func (b Box) CutTop(h int) (top, rest Box) {
	if h <= 0 {
		return Box{R: cellbuf.Rectangle{Min: b.R.Min, Max: cellbuf.Pos(b.R.Max.X, b.R.Min.Y)}}, b
	}
	if h >= b.R.Dy() {
		return b, Box{R: cellbuf.Rectangle{Min: cellbuf.Pos(b.R.Min.X, b.R.Max.Y), Max: b.R.Max}}
	}

	splitY := b.R.Min.Y + h
	top = Box{R: cellbuf.Rectangle{
		Min: b.R.Min,
		Max: cellbuf.Pos(b.R.Max.X, splitY),
	}}
	rest = Box{R: cellbuf.Rectangle{
		Min: cellbuf.Pos(b.R.Min.X, splitY),
		Max: b.R.Max,
	}}
	return
}

// CutBottom cuts h cells from the bottom, returning the rest and the bottom box.
func (b Box) CutBottom(h int) (rest, bottom Box) {
	if h <= 0 {
		return b, Box{R: cellbuf.Rectangle{Min: cellbuf.Pos(b.R.Min.X, b.R.Max.Y), Max: b.R.Max}}
	}
	if h >= b.R.Dy() {
		return Box{R: cellbuf.Rectangle{Min: b.R.Min, Max: cellbuf.Pos(b.R.Max.X, b.R.Min.Y)}}, b
	}

	splitY := b.R.Max.Y - h
	rest = Box{R: cellbuf.Rectangle{
		Min: b.R.Min,
		Max: cellbuf.Pos(b.R.Max.X, splitY),
	}}
	bottom = Box{R: cellbuf.Rectangle{
		Min: cellbuf.Pos(b.R.Min.X, splitY),
		Max: b.R.Max,
	}}
	return
}

// CutLeft cuts w cells from the left, returning the left box and the rest.
func (b Box) CutLeft(w int) (left, rest Box) {
	if w <= 0 {
		return Box{R: cellbuf.Rectangle{Min: b.R.Min, Max: cellbuf.Pos(b.R.Min.X, b.R.Max.Y)}}, b
	}
	if w >= b.R.Dx() {
		return b, Box{R: cellbuf.Rectangle{Min: cellbuf.Pos(b.R.Max.X, b.R.Min.Y), Max: b.R.Max}}
	}

	splitX := b.R.Min.X + w
	left = Box{R: cellbuf.Rectangle{
		Min: b.R.Min,
		Max: cellbuf.Pos(splitX, b.R.Max.Y),
	}}
	rest = Box{R: cellbuf.Rectangle{
		Min: cellbuf.Pos(splitX, b.R.Min.Y),
		Max: b.R.Max,
	}}
	return
}

// CutRight cuts w cells from the right, returning the rest and the right box.
func (b Box) CutRight(w int) (rest, right Box) {
	if w <= 0 {
		return b, Box{R: cellbuf.Rectangle{Min: cellbuf.Pos(b.R.Max.X, b.R.Min.Y), Max: b.R.Max}}
	}
	if w >= b.R.Dx() {
		return Box{R: cellbuf.Rectangle{Min: b.R.Min, Max: cellbuf.Pos(b.R.Min.X, b.R.Max.Y)}}, b
	}

	splitX := b.R.Max.X - w
	rest = Box{R: cellbuf.Rectangle{
		Min: b.R.Min,
		Max: cellbuf.Pos(splitX, b.R.Max.Y),
	}}
	right = Box{R: cellbuf.Rectangle{
		Min: cellbuf.Pos(splitX, b.R.Min.Y),
		Max: b.R.Max,
	}}
	return
}

// Center returns a box of size w×h centered within this box.
// If the requested size is larger than available space, it's clamped.
func (b Box) Center(w, h int) Box {
	availWidth := b.R.Dx()
	availHeight := b.R.Dy()

	// Clamp to available space
	if w > availWidth {
		w = availWidth
	}
	if h > availHeight {
		h = availHeight
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}

	// Calculate offsets to center
	offsetX := (availWidth - w) / 2
	offsetY := (availHeight - h) / 2

	return Box{R: cellbuf.Rectangle{
		Min: cellbuf.Pos(b.R.Min.X+offsetX, b.R.Min.Y+offsetY),
		Max: cellbuf.Pos(b.R.Min.X+offsetX+w, b.R.Min.Y+offsetY+h),
	}}
}
