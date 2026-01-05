package render

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
)

// DisplayList holds all rendering operations for a frame.
// Operations are accumulated during the layout/render pass,
// then executed in order (Draw calls by Z, then Effects by Z).
type DisplayList struct {
	draws        []Draw
	effects      []Effect
	interactions []InteractionOp
}

// NewDisplayList creates a new empty display list.
func NewDisplayList() *DisplayList {
	return &DisplayList{
		draws:        make([]Draw, 0, 16),
		effects:      make([]Effect, 0, 8),
		interactions: make([]InteractionOp, 0, 8),
	}
}

// AddDraw adds a Draw to the display list.
func (dl *DisplayList) AddDraw(rect cellbuf.Rectangle, content string, z int) {
	dl.draws = append(dl.draws, Draw{
		Rect:    rect,
		Content: content,
		Z:       z,
	})
}

// AddEffect adds a custom Effect to the display list.
// This is the generic method that accepts any Effect implementation.
func (dl *DisplayList) AddEffect(effect Effect) {
	dl.effects = append(dl.effects, effect)
}

// AddReverse adds a ReverseEffect (reverses foreground/background colors).
func (dl *DisplayList) AddReverse(rect cellbuf.Rectangle, z int) {
	dl.AddEffect(ReverseEffect{Rect: rect, Z: z})
}

// AddDim adds a DimEffect (dims the content).
func (dl *DisplayList) AddDim(rect cellbuf.Rectangle, z int) {
	dl.AddEffect(DimEffect{Rect: rect, Z: z})
}

// AddUnderline adds an UnderlineEffect.
func (dl *DisplayList) AddUnderline(rect cellbuf.Rectangle, z int) {
	dl.AddEffect(UnderlineEffect{Rect: rect, Z: z})
}

// AddBold adds a BoldEffect.
func (dl *DisplayList) AddBold(rect cellbuf.Rectangle, z int) {
	dl.AddEffect(BoldEffect{Rect: rect, Z: z})
}

// AddStrike adds a StrikeEffect (strikethrough).
func (dl *DisplayList) AddStrike(rect cellbuf.Rectangle, z int) {
	dl.AddEffect(StrikeEffect{Rect: rect, Z: z})
}

// AddHighlight adds a HighlightEffect.
func (dl *DisplayList) AddHighlight(rect cellbuf.Rectangle, style lipgloss.Style, z int) {
	dl.AddEffect(HighlightEffect{Rect: rect, Style: style, Z: z})
}

// AddInteraction adds an InteractionOp to the display list.
func (dl *DisplayList) AddInteraction(rect cellbuf.Rectangle, msg tea.Msg, typ InteractionType, z int) {
	dl.interactions = append(dl.interactions, InteractionOp{
		Rect: rect,
		Msg:  msg,
		Type: typ,
		Z:    z,
	})
}

// Clear removes all operations from the display list.
// Useful for reusing a DisplayList across frames.
func (dl *DisplayList) Clear() {
	dl.draws = dl.draws[:0]
	dl.effects = dl.effects[:0]
	dl.interactions = dl.interactions[:0]
}

// Render executes all operations in the display list to the given cellbuf.
// Order of execution:
// 1. Draw sorted by Z-index (low to high)
// 2. Effects sorted by Z-index (low to high)
func (dl *DisplayList) Render(buf *cellbuf.Buffer) {
	// Sort Draw calls by Z-index (stable sort maintains insertion order for equal Z)
	sort.SliceStable(dl.draws, func(i, j int) bool {
		return dl.draws[i].Z < dl.draws[j].Z
	})

	// Render all Draw calls
	for _, op := range dl.draws {
		cellbuf.SetContentRect(buf, op.Content, op.Rect)
	}

	// Sort Effects by Z-index
	sort.SliceStable(dl.effects, func(i, j int) bool {
		return dl.effects[i].GetZ() < dl.effects[j].GetZ()
	})

	// Apply all Effects
	for _, effect := range dl.effects {
		effect.Apply(buf)
	}
}

// RenderToString is a convenience method that renders to a new buffer
// and returns the final string output.
func (dl *DisplayList) RenderToString(width, height int) string {
	buf := cellbuf.NewBuffer(width, height)
	dl.Render(buf)
	return cellbuf.Render(buf)
}

// DrawList returns a copy of all Draw calls (useful for debugging/inspection)
func (dl *DisplayList) DrawList() []Draw {
	result := make([]Draw, len(dl.draws))
	copy(result, dl.draws)
	return result
}

// EffectsList returns a copy of all Effects (useful for debugging/inspection)
func (dl *DisplayList) EffectsList() []Effect {
	result := make([]Effect, len(dl.effects))
	copy(result, dl.effects)
	return result
}

// InteractionsList returns all interactions sorted by Z-index (highest first for priority).
func (dl *DisplayList) InteractionsList() []InteractionOp {
	sorted := make([]InteractionOp, len(dl.interactions))
	copy(sorted, dl.interactions)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Z > sorted[j].Z // Higher Z = higher priority
	})
	return sorted
}

// Merge adds all operations from another DisplayList into this one.
func (dl *DisplayList) Merge(other *DisplayList) {
	dl.draws = append(dl.draws, other.draws...)
	dl.effects = append(dl.effects, other.effects...)
	dl.interactions = append(dl.interactions, other.interactions...)
}

// Len returns the total number of operations in the display list
func (dl *DisplayList) Len() int {
	return len(dl.draws) + len(dl.effects) + len(dl.interactions)
}
