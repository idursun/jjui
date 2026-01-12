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

// ScrollDeltaCarrier is an interface for messages that carry scroll delta information.
// The ProcessMouseEvent function will set the Delta field for scroll interactions.
type ScrollDeltaCarrier interface {
	SetDelta(delta int) tea.Msg
}

// ProcessMouseEvent matches a mouse event against interactions and returns the associated message.
// Interactions are expected to be sorted by Z-index (highest first) for proper priority handling.
// For scroll interactions, if the message implements ScrollDeltaCarrier, the delta will be set.
func ProcessMouseEvent(interactions []InteractionOp, msg tea.MouseMsg) tea.Msg {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// Find highest-Z clickable region containing this point
			for _, interaction := range interactions {
				if interaction.Type&InteractionClick == 0 {
					continue
				}
				if msg.X >= interaction.Rect.Min.X && msg.X < interaction.Rect.Max.X &&
					msg.Y >= interaction.Rect.Min.Y && msg.Y < interaction.Rect.Max.Y {
					return interaction.Msg
				}
			}
		}

		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			delta := -3
			if msg.Button == tea.MouseButtonWheelDown {
				delta = 3
			}

			// Find scrollable region containing this point
			for _, interaction := range interactions {
				if interaction.Type&InteractionScroll == 0 {
					continue
				}
				if msg.X >= interaction.Rect.Min.X && msg.X < interaction.Rect.Max.X &&
					msg.Y >= interaction.Rect.Min.Y && msg.Y < interaction.Rect.Max.Y {
					// If the message can carry delta, set it
					if carrier, ok := interaction.Msg.(ScrollDeltaCarrier); ok {
						return carrier.SetDelta(delta)
					}
					return interaction.Msg
				}
			}
		}
	}

	return nil
}
