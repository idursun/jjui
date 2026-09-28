package render

import (
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/layout"
)

// InteractionType defines what kinds of input an interactive region responds to.
// Multiple types can be combined using bitwise OR.
type InteractionType int

const (
	InteractionClick InteractionType = 1 << iota
	InteractionScroll
	InteractionDrag
)

// InteractionOp represents an interactive region that responds to input.
type InteractionOp struct {
	Rect  layout.Rectangle           // The interactive area (absolute coordinates)
	Msg   tea.Msg                    // Message to send (static)
	MsgFn func(tea.MouseMsg) tea.Msg // Optional message factory.
	Type  InteractionType            // What kind of interaction this supports
	Z     int                        // Z-index for overlapping regions (higher = priority)
}

// ScrollDeltaCarrier is an interface for messages that carry scroll delta information.
// The ProcessMouseEvent function will set the Delta field for scroll interactions.
type ScrollDeltaCarrier interface {
	SetDelta(delta int, horizontal bool) tea.Msg
}

// DragStartCarrier is an interface for messages that carry drag start coordinates.
// The ProcessMouseEvent function will set the drag start position for drag interactions.
type DragStartCarrier interface {
	SetDragStart(x, y int) tea.Msg
}

func processMouseEvent(interactions []interactionOp, msg tea.MouseMsg) (tea.Msg, bool) {
	mouse := msg.Mouse()
	pos := layout.Pos(mouse.X, mouse.Y)
	switch msg.(type) {
	case tea.MouseClickMsg:
		if mouse.Button != tea.MouseLeft {
			return nil, false
		}
		// Find highest-Z draggable region containing this point
		if interaction, ok := findInteraction(interactions, pos, InteractionDrag); ok {
			if interaction.MsgFn != nil {
				return interaction.MsgFn(msg), true
			}
			if carrier, ok := interaction.Msg.(DragStartCarrier); ok {
				return carrier.SetDragStart(mouse.X, mouse.Y), true
			}
			return interaction.Msg, true
		}
		// Find highest-Z clickable region containing this point
		if interaction, ok := findInteraction(interactions, pos, InteractionClick); ok {
			if interaction.MsgFn != nil {
				return interaction.MsgFn(msg), true
			}
			return interaction.Msg, true
		}
	case tea.MouseWheelMsg:
		var delta int
		var horizontal bool
		switch mouse.Button {
		case tea.MouseWheelUp:
			delta = -3
		case tea.MouseWheelDown:
			delta = 3
		case tea.MouseWheelLeft:
			delta, horizontal = -3, true
		case tea.MouseWheelRight:
			delta, horizontal = 3, true
		default:
			return nil, false
		}
		if interaction, ok := findInteraction(interactions, pos, InteractionScroll); ok {
			if carrier, ok := interaction.Msg.(ScrollDeltaCarrier); ok {
				return carrier.SetDelta(delta, horizontal), true
			}
			return interaction.Msg, true
		}
	}

	return nil, false
}

// findInteraction returns the first interaction of the given type containing pos.
// Interactions are expected to be sorted by priority.
func findInteraction(interactions []interactionOp, pos layout.Position, typ InteractionType) (interactionOp, bool) {
	for _, interaction := range interactions {
		if interaction.Type&typ != 0 && pos.In(interaction.Rect) {
			return interaction, true
		}
	}
	return interactionOp{}, false
}
