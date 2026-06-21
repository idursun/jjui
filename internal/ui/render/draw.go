package render

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/idursun/jjui/internal/ui/layout"
)

// Draw represents a content rendering operation.
// DrawOps are rendered first, sorted by Z-index (lower values render first).
type Draw struct {
	Rect    layout.Rectangle // The area to draw in
	Content string           // Rendered ANSI string (from lipgloss, etc.)
	Z       int              // Z-index for layering (lower = back, higher = front)
	Options DrawOptions
}

type DrawOptions struct {
	PreserveBackground bool
}

type DrawOption func(*DrawOptions)

func PreserveBackground() DrawOption {
	return func(opts *DrawOptions) {
		opts.PreserveBackground = true
	}
}

// ReplayTerminalOutput replays captured command output through an offscreen
// terminal buffer. Applied to the rendering of flash message and command
// history
func ReplayTerminalOutput(content string) string {
	width := 1
	for field := range strings.FieldsFuncSeq(content, func(r rune) bool {
		return r == '\r' || r == '\n'
	}) {
		// finds the max width of the content
		width = max(width, StringWidth(field))
	}

	height := strings.Count(content, "\n") + 1
	// builds an offscreen terminal and draws the content
	buf := NewScreenBuffer(width, height)
	uv.NewStyledString(content).Draw(buf, layout.Rect(0, 0, width, height))
	return buf.Render()
}
