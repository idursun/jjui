package view

// Constraints provides a set of predefined constraints for common layout patterns

// Grow creates a constraint that will grow to fill available space
func Grow(factor float64) Constraint {
	return Constraint{
		GrowthFactor: factor,
		FitContent:   false,
		MinSize:      0,
		MaxSize:      0,
		FixedSize:    0,
	}
}

// Fixed creates a constraint with a fixed size
func Fixed(size int) Constraint {
	return Constraint{
		GrowthFactor: 0,
		FitContent:   false,
		MinSize:      0,
		MaxSize:      0,
		FixedSize:    size,
	}
}

// FitContent creates a constraint that sizes according to content
func FitContent() Constraint {
	return Constraint{
		GrowthFactor: 0,
		FitContent:   true,
		MinSize:      0,
		MaxSize:      0,
		FixedSize:    0,
	}
}

// WithMinSize sets a minimum size on a constraint
func (c Constraint) WithMinSize(size int) Constraint {
	c.MinSize = size
	return c
}

// WithMaxSize sets a maximum size on a constraint
func (c Constraint) WithMaxSize(size int) Constraint {
	c.MaxSize = size
	return c
}

// Equal splits space equally between views
func Equal() Constraint {
	return Grow(1.0)
}

// Ratio creates a constraint with a specific ratio of the available space
func Ratio(ratio float64) Constraint {
	return Grow(ratio)
}
