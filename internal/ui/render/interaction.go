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
	InteractionHover
)

// InteractionOp represents an interactive region that responds to input.
type InteractionOp struct {
	Rect layout.Rectangle // The interactive area (absolute coordinates)
	Msg  tea.Msg          // Message to send
	Type InteractionType  // What kind of interaction this supports
	Z    int              // Z-index for overlapping regions (higher = priority)
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

type interactionMatcher func(interactionOp) bool

func processMouseEvent(interactions []interactionOp, msg tea.MouseMsg, match interactionMatcher) (tea.Msg, bool) {
	mouse := msg.Mouse()
	switch msg.(type) {
	case tea.MouseClickMsg:
		if mouse.Button == tea.MouseLeft {
			// Find highest-Z draggable region containing this point
			for _, interaction := range interactions {
				if !match(interaction) || interaction.Type&InteractionDrag == 0 {
					continue
				}
				if mouse.X >= interaction.Rect.Min.X && mouse.X < interaction.Rect.Max.X &&
					mouse.Y >= interaction.Rect.Min.Y && mouse.Y < interaction.Rect.Max.Y {
					if carrier, ok := interaction.Msg.(DragStartCarrier); ok {
						return carrier.SetDragStart(mouse.X, mouse.Y), true
					}
					return interaction.Msg, true
				}
			}

			// Find highest-Z clickable region containing this point
			for _, interaction := range interactions {
				if !match(interaction) || interaction.Type&InteractionClick == 0 {
					continue
				}
				if mouse.X >= interaction.Rect.Min.X && mouse.X < interaction.Rect.Max.X &&
					mouse.Y >= interaction.Rect.Min.Y && mouse.Y < interaction.Rect.Max.Y {
					return interaction.Msg, true
				}
			}
		}
	case tea.MouseWheelMsg:
		switch mouse.Button {
		case tea.MouseWheelUp, tea.MouseWheelDown:
			delta := -3
			if mouse.Button == tea.MouseWheelDown {
				delta = 3
			}
			for _, interaction := range interactions {
				if !match(interaction) || interaction.Type&InteractionScroll == 0 {
					continue
				}
				if mouse.X >= interaction.Rect.Min.X && mouse.X < interaction.Rect.Max.X &&
					mouse.Y >= interaction.Rect.Min.Y && mouse.Y < interaction.Rect.Max.Y {
					if carrier, ok := interaction.Msg.(ScrollDeltaCarrier); ok {
						return carrier.SetDelta(delta, false), true
					}
					return interaction.Msg, true
				}
			}
		case tea.MouseWheelLeft, tea.MouseWheelRight:
			delta := -3
			if mouse.Button == tea.MouseWheelRight {
				delta = 3
			}
			for _, interaction := range interactions {
				if !match(interaction) || interaction.Type&InteractionScroll == 0 {
					continue
				}
				if mouse.X >= interaction.Rect.Min.X && mouse.X < interaction.Rect.Max.X &&
					mouse.Y >= interaction.Rect.Min.Y && mouse.Y < interaction.Rect.Max.Y {
					if carrier, ok := interaction.Msg.(ScrollDeltaCarrier); ok {
						return carrier.SetDelta(delta, true), true
					}
					return nil, true
				}
			}
		}
	}

	return nil, false
}
