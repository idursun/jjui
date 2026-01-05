package render

import (
	"github.com/charmbracelet/x/cellbuf"
)

// Draw represents a content rendering operation.
// DrawOps are rendered first, sorted by Z-index (lower values render first).
type Draw struct {
	Rect    cellbuf.Rectangle // The area to draw in
	Content string            // Rendered ANSI string (from lipgloss, etc.)
	Z       int               // Z-index for layering (lower = back, higher = front)
}
