package render

import (
	"github.com/idursun/jjui/internal/ui/layout"
)

// Draw represents a content rendering operation.
// DrawOps are rendered first, sorted by Z-index (lower values render first).
type Draw struct {
	Rect    layout.Rectangle // The area to draw in
	Content string           // Rendered ANSI string (from lipgloss, etc.)
	Z       int              // Z-index for layering (lower = back, higher = front)
}
