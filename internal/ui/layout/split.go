package layout

// Split represents a resizable split between two areas.
// It tracks the current percentage and provides methods for layout and drag handling.
type Split struct {
	Percentage float64 // 0-100, percentage for secondary panel
	Vertical   bool    // true = top/bottom, false = left/right
	MinPercent float64 // Minimum percentage (e.g., 10)
	MaxPercent float64 // Maximum percentage (e.g., 95)
}

// NewSplit creates a new Split with the given initial percentage and orientation.
func NewSplit(percentage float64, vertical bool) *Split {
	s := &Split{
		Percentage: percentage,
		Vertical:   vertical,
		MinPercent: 10,
		MaxPercent: 95,
	}
	s.clamp()
	return s
}

// Apply splits the box according to the current percentage.
// Returns (main, secondary) boxes.
// If Vertical: main is top, secondary is bottom.
// If Horizontal: main is left, secondary is right.
func (s *Split) Apply(box Box) (main, secondary Box) {
	mainPct := 100 - s.Percentage
	if s.Vertical {
		boxes := box.V(Percent(mainPct), Fill(1))
		return boxes[0], boxes[1]
	}
	boxes := box.H(Percent(mainPct), Fill(1))
	return boxes[0], boxes[1]
}

// DragTo calculates the new percentage from the given coordinates within the box.
// For vertical splits, y position determines the percentage.
// For horizontal splits, x position determines the percentage.
// Returns true if the percentage changed.
func (s *Split) DragTo(box Box, x, y int) bool {
	oldPct := s.Percentage

	if s.Vertical {
		// Vertical split: percentage is distance from bottom
		totalHeight := box.R.Dy()
		if totalHeight <= 0 {
			return false
		}
		distanceFromBottom := box.R.Max.Y - y
		s.Percentage = float64(distanceFromBottom*100) / float64(totalHeight)
	} else {
		// Horizontal split: percentage is distance from right
		totalWidth := box.R.Dx()
		if totalWidth <= 0 {
			return false
		}
		distanceFromRight := box.R.Max.X - x
		s.Percentage = float64(distanceFromRight*100) / float64(totalWidth)
	}

	s.clamp()
	return s.Percentage != oldPct
}

// Expand increases the secondary panel size by delta percentage.
func (s *Split) Expand(delta float64) {
	s.Percentage += delta
	s.clamp()
}

// Shrink decreases the secondary panel size by delta percentage.
func (s *Split) Shrink(delta float64) {
	s.Percentage -= delta
	s.clamp()
}

// SetVertical changes the split orientation.
func (s *Split) SetVertical(vertical bool) {
	s.Vertical = vertical
}

// clamp ensures percentage stays within bounds.
func (s *Split) clamp() {
	if s.Percentage < s.MinPercent {
		s.Percentage = s.MinPercent
	}
	if s.Percentage > s.MaxPercent {
		s.Percentage = s.MaxPercent
	}
}
