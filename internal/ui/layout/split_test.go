package layout

import (
	"testing"

	"github.com/charmbracelet/x/cellbuf"
)

func TestNewSplit(t *testing.T) {
	s := NewSplit(30, true)

	if s.Percentage != 30 {
		t.Errorf("Percentage = %f, want 30", s.Percentage)
	}
	if !s.Vertical {
		t.Error("Vertical = false, want true")
	}
	if s.MinPercent != 10 {
		t.Errorf("MinPercent = %f, want 10", s.MinPercent)
	}
	if s.MaxPercent != 95 {
		t.Errorf("MaxPercent = %f, want 95", s.MaxPercent)
	}
}

func TestNewSplit_Clamping(t *testing.T) {
	tests := []struct {
		name       string
		percentage float64
		want       float64
	}{
		{"below_min", 5, 10},
		{"above_max", 99, 95},
		{"within_bounds", 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSplit(tt.percentage, true)
			if s.Percentage != tt.want {
				t.Errorf("Percentage = %f, want %f", s.Percentage, tt.want)
			}
		})
	}
}

func TestSplit_Apply_Vertical(t *testing.T) {
	s := NewSplit(30, true) // 30% for secondary (bottom)
	box := NewBox(cellbuf.Rect(0, 0, 100, 100))

	main, secondary := s.Apply(box)

	// Main should be 70% (top)
	if main.R.Dy() != 70 {
		t.Errorf("main height = %d, want 70", main.R.Dy())
	}
	if main.R.Min.Y != 0 {
		t.Errorf("main starts at Y=%d, want 0", main.R.Min.Y)
	}

	// Secondary should be 30% (bottom)
	if secondary.R.Dy() != 30 {
		t.Errorf("secondary height = %d, want 30", secondary.R.Dy())
	}
	if secondary.R.Min.Y != 70 {
		t.Errorf("secondary starts at Y=%d, want 70", secondary.R.Min.Y)
	}
}

func TestSplit_Apply_Horizontal(t *testing.T) {
	s := NewSplit(40, false) // 40% for secondary (right)
	box := NewBox(cellbuf.Rect(0, 0, 100, 50))

	main, secondary := s.Apply(box)

	// Main should be 60% (left)
	if main.R.Dx() != 60 {
		t.Errorf("main width = %d, want 60", main.R.Dx())
	}
	if main.R.Min.X != 0 {
		t.Errorf("main starts at X=%d, want 0", main.R.Min.X)
	}

	// Secondary should be 40% (right)
	if secondary.R.Dx() != 40 {
		t.Errorf("secondary width = %d, want 40", secondary.R.Dx())
	}
	if secondary.R.Min.X != 60 {
		t.Errorf("secondary starts at X=%d, want 60", secondary.R.Min.X)
	}
}

func TestSplit_DragTo_Vertical(t *testing.T) {
	s := NewSplit(50, true)
	box := NewBox(cellbuf.Rect(0, 0, 100, 100))

	// Drag to y=70 (30 from bottom = 30%)
	changed := s.DragTo(box, 50, 70)

	if !changed {
		t.Error("DragTo should return true when percentage changes")
	}
	if s.Percentage != 30 {
		t.Errorf("Percentage = %f, want 30", s.Percentage)
	}
}

func TestSplit_DragTo_Horizontal(t *testing.T) {
	s := NewSplit(50, false)
	box := NewBox(cellbuf.Rect(0, 0, 100, 50))

	// Drag to x=60 (40 from right = 40%)
	changed := s.DragTo(box, 60, 25)

	if !changed {
		t.Error("DragTo should return true when percentage changes")
	}
	if s.Percentage != 40 {
		t.Errorf("Percentage = %f, want 40", s.Percentage)
	}
}

func TestSplit_DragTo_Clamping(t *testing.T) {
	s := NewSplit(50, true)
	box := NewBox(cellbuf.Rect(0, 0, 100, 100))

	// Drag to y=2 (98% from bottom, should clamp to 95%)
	s.DragTo(box, 50, 2)
	if s.Percentage != 95 {
		t.Errorf("Percentage = %f, want 95 (clamped)", s.Percentage)
	}

	// Drag to y=95 (5% from bottom, should clamp to 10%)
	s.DragTo(box, 50, 95)
	if s.Percentage != 10 {
		t.Errorf("Percentage = %f, want 10 (clamped)", s.Percentage)
	}
}

func TestSplit_DragTo_ZeroSize(t *testing.T) {
	s := NewSplit(50, true)
	box := NewBox(cellbuf.Rect(0, 0, 100, 0)) // zero height

	changed := s.DragTo(box, 50, 0)

	if changed {
		t.Error("DragTo should return false for zero-size box")
	}
	if s.Percentage != 50 {
		t.Errorf("Percentage should be unchanged, got %f", s.Percentage)
	}
}

func TestSplit_DragTo_NoChange(t *testing.T) {
	s := NewSplit(50, true)
	box := NewBox(cellbuf.Rect(0, 0, 100, 100))

	// Drag to exactly 50%
	changed := s.DragTo(box, 50, 50)

	if changed {
		t.Error("DragTo should return false when percentage doesn't change")
	}
}

func TestSplit_Expand(t *testing.T) {
	s := NewSplit(30, true)

	s.Expand(10)
	if s.Percentage != 40 {
		t.Errorf("Percentage after Expand = %f, want 40", s.Percentage)
	}

	// Expand past max should clamp
	s.Expand(100)
	if s.Percentage != 95 {
		t.Errorf("Percentage after overflow Expand = %f, want 95", s.Percentage)
	}
}

func TestSplit_Shrink(t *testing.T) {
	s := NewSplit(50, true)

	s.Shrink(10)
	if s.Percentage != 40 {
		t.Errorf("Percentage after Shrink = %f, want 40", s.Percentage)
	}

	// Shrink past min should clamp
	s.Shrink(100)
	if s.Percentage != 10 {
		t.Errorf("Percentage after underflow Shrink = %f, want 10", s.Percentage)
	}
}

func TestSplit_SetVertical(t *testing.T) {
	s := NewSplit(50, true)

	s.SetVertical(false)
	if s.Vertical {
		t.Error("Vertical should be false after SetVertical(false)")
	}

	s.SetVertical(true)
	if !s.Vertical {
		t.Error("Vertical should be true after SetVertical(true)")
	}
}

func TestSplit_Apply_OffsetBox(t *testing.T) {
	s := NewSplit(30, true)
	box := NewBox(cellbuf.Rect(10, 20, 100, 80)) // offset box

	main, secondary := s.Apply(box)

	// Check X coordinates are preserved
	if main.R.Min.X != 10 || main.R.Max.X != 110 {
		t.Errorf("main X range = [%d, %d], want [10, 110]", main.R.Min.X, main.R.Max.X)
	}
	if secondary.R.Min.X != 10 || secondary.R.Max.X != 110 {
		t.Errorf("secondary X range = [%d, %d], want [10, 110]", secondary.R.Min.X, secondary.R.Max.X)
	}

	// Check Y split is correct
	// 80 height, 70% main = 56, 30% secondary = 24
	if main.R.Dy() != 56 {
		t.Errorf("main height = %d, want 56", main.R.Dy())
	}
	if secondary.R.Dy() != 24 {
		t.Errorf("secondary height = %d, want 24", secondary.R.Dy())
	}
}
