package render

import (
	"sort"

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
	SetDelta(delta int, horizontal bool) tea.Msg
}

// DragStartCarrier is an interface for messages that carry drag start coordinates.
// The ProcessMouseEvent function will set the drag start position for drag interactions.
type DragStartCarrier interface {
	SetDragStart(x, y int) tea.Msg
}

type interactionMatcher func(interactionOp) bool

func processMouseEvent(interactions []interactionOp, msg tea.MouseMsg, match interactionMatcher) (tea.Msg, bool) {
	if msg.Action != tea.MouseActionPress {
		return nil, false
	}

	if msg.Button == tea.MouseButtonLeft {
		// Find highest-Z draggable region containing this point
		for _, interaction := range interactions {
			if !match(interaction) || interaction.Type&InteractionDrag == 0 {
				continue
			}
			if msg.X >= interaction.Rect.Min.X && msg.X < interaction.Rect.Max.X &&
				msg.Y >= interaction.Rect.Min.Y && msg.Y < interaction.Rect.Max.Y {
				if carrier, ok := interaction.Msg.(DragStartCarrier); ok {
					return carrier.SetDragStart(msg.X, msg.Y), true
				}
				return interaction.Msg, true
			}
		}

		// Find highest-Z clickable region containing this point
		for _, interaction := range interactions {
			if !match(interaction) || interaction.Type&InteractionClick == 0 {
				continue
			}
			if msg.X >= interaction.Rect.Min.X && msg.X < interaction.Rect.Max.X &&
				msg.Y >= interaction.Rect.Min.Y && msg.Y < interaction.Rect.Max.Y {
				return interaction.Msg, true
			}
		}
	}

	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		delta := -3
		if msg.Button == tea.MouseButtonWheelDown {
			delta = 3
		}
		for _, interaction := range interactions {
			if !match(interaction) || interaction.Type&InteractionScroll == 0 {
				continue
			}
			if msg.X >= interaction.Rect.Min.X && msg.X < interaction.Rect.Max.X &&
				msg.Y >= interaction.Rect.Min.Y && msg.Y < interaction.Rect.Max.Y {
				if carrier, ok := interaction.Msg.(ScrollDeltaCarrier); ok {
					return carrier.SetDelta(delta, false), true
				}
				return interaction.Msg, true
			}
		}
	}

	if msg.Button == tea.MouseButtonWheelLeft || msg.Button == tea.MouseButtonWheelRight {
		delta := -3
		if msg.Button == tea.MouseButtonWheelRight {
			delta = 3
		}
		for _, interaction := range interactions {
			if !match(interaction) || interaction.Type&InteractionScroll == 0 {
				continue
			}
			if msg.X >= interaction.Rect.Min.X && msg.X < interaction.Rect.Max.X &&
				msg.Y >= interaction.Rect.Min.Y && msg.Y < interaction.Rect.Max.Y {
				if carrier, ok := interaction.Msg.(ScrollDeltaCarrier); ok {
					return carrier.SetDelta(delta, true), true
				}
				return nil, true
			}
		}
	}

	return nil, false
}

// ProcessMouseEvent matches a mouse event against interactions and returns the associated message.
// Interactions are expected to be sorted by Z-index (highest first) for proper priority handling.
// For scroll interactions, if the message implements ScrollDeltaCarrier, the delta will be set.
func ProcessMouseEvent(interactions []InteractionOp, msg tea.MouseMsg) tea.Msg {
	wrapped := make([]interactionOp, len(interactions))
	for i, interaction := range interactions {
		wrapped[i] = interactionOp{
			InteractionOp: interaction,
			order:         i,
		}
	}
	msgResult, handled := processMouseEvent(wrapped, msg, func(interactionOp) bool {
		return true
	})
	if handled {
		return msgResult
	}
	return nil
}

// ProcessMouseEventWithWindows routes a mouse event through window scopes.
func ProcessMouseEventWithWindows(interactions []interactionOp, windows []windowOp, msg tea.MouseMsg) (tea.Msg, bool) {
	windowID, windowHit := topWindowAt(windows, msg.X, msg.Y)
	if msg.Action != tea.MouseActionPress {
		return nil, windowHit
	}

	sorted := make([]interactionOp, len(interactions))
	copy(sorted, interactions)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Z != sorted[j].Z {
			return sorted[i].Z > sorted[j].Z
		}
		return sorted[i].order < sorted[j].order
	})

	msgResult, handled := processMouseEvent(sorted, msg, func(interaction interactionOp) bool {
		return windowMatch(interaction.windowID, windowID, windowHit)
	})
	if handled {
		return msgResult, true
	}
	return nil, windowHit
}

func topWindowAt(windows []windowOp, x, y int) (int, bool) {
	if len(windows) == 0 {
		return 0, false
	}
	sorted := make([]windowOp, len(windows))
	copy(sorted, windows)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Z != sorted[j].Z {
			return sorted[i].Z > sorted[j].Z
		}
		return sorted[i].Order > sorted[j].Order
	})
	for _, win := range sorted {
		if x >= win.Rect.Min.X && x < win.Rect.Max.X &&
			y >= win.Rect.Min.Y && y < win.Rect.Max.Y {
			return win.ID, true
		}
	}
	return 0, false
}

func windowMatch(interactionWindowID, windowID int, windowHit bool) bool {
	if windowHit {
		return interactionWindowID == windowID
	}
	return interactionWindowID == 0
}
