package render

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
)

// InteractionType defines what kinds of input an interactive region responds to.
// Multiple types can be combined using bitwise OR.
type InteractionType int

const (
	InteractionClick InteractionType = 1 << iota
	InteractionScroll
	InteractionDrag
	InteractionHover
)

// InteractionOp represents an interactive region that responds to input.
type InteractionOp struct {
	Rect cellbuf.Rectangle // The interactive area (absolute coordinates)
	Msg  tea.Msg           // Message to send
	Type InteractionType   // What kind of interaction this supports
	Z    int               // Z-index for overlapping regions (higher = priority)
}
