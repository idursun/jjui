package render

import (
	"sort"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/idursun/jjui/internal/ui/layout"
)

// DisplayContext holds all rendering operations for a frame.
// Operations are accumulated during the layout/render pass,
// then executed in order by batch and Z-index.
type DisplayContext struct {
	draws        []drawOp
	effects      []effectOp
	interactions []interactionOp
	orderCounter int
}

// NewDisplayContext creates a new empty display context.
func NewDisplayContext() *DisplayContext {
	return &DisplayContext{
		draws:        make([]drawOp, 0, 16),
		effects:      make([]effectOp, 0, 8),
		interactions: make([]interactionOp, 0, 8),
	}
}

func (dl *DisplayContext) nextOrder() int {
	dl.orderCounter++
	return dl.orderCounter
}

// AddBackdrop swallows click/scroll input in a region.
func (dl *DisplayContext) AddBackdrop(rect layout.Rectangle, z int) {
	dl.AddInteraction(rect, nil, InteractionClick|InteractionScroll, z)
}

// AddDraw adds a Draw to the display context.
func (dl *DisplayContext) AddDraw(rect layout.Rectangle, content string, z int) {
	dl.draws = append(dl.draws, drawOp{
		Draw: Draw{
			Rect:    rect,
			Content: content,
			Z:       z,
		},
		order: dl.nextOrder(),
	})
}

// AddFill fills a rectangle with the provided rune and style.
func (dl *DisplayContext) AddFill(rect layout.Rectangle, ch rune, style lipgloss.Style, z int) {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	dl.AddEffect(FillEffect{
		Rect:  rect,
		Char:  ch,
		Style: lipglossToStyle(style),
		Z:     z,
	})
}

// AddEffect adds a custom Effect to the display context.
// This is the generic method that accepts any Effect implementation.
func (dl *DisplayContext) AddEffect(effect Effect) {
	dl.effects = append(dl.effects, effectOp{
		effect: effect,
		order:  dl.nextOrder(),
		z:      effect.GetZ(),
	})
}

// AddReverse adds a ReverseEffect (reverses foreground/background colors).
func (dl *DisplayContext) AddReverse(rect layout.Rectangle, z int) {
	dl.AddEffect(ReverseEffect{Rect: rect, Z: z})
}

// AddDim adds a DimEffect (dims the content).
func (dl *DisplayContext) AddDim(rect layout.Rectangle, z int) {
	dl.AddEffect(DimEffect{Rect: rect, Z: z})
}

// AddUnderline adds an UnderlineEffect.
func (dl *DisplayContext) AddUnderline(rect layout.Rectangle, z int) {
	dl.AddEffect(UnderlineEffect{Rect: rect, Z: z})
}

// AddBold adds a BoldEffect.
func (dl *DisplayContext) AddBold(rect layout.Rectangle, z int) {
	dl.AddEffect(BoldEffect{Rect: rect, Z: z})
}

// AddStrike adds a StrikeEffect (strikethrough).
func (dl *DisplayContext) AddStrike(rect layout.Rectangle, z int) {
	dl.AddEffect(StrikeEffect{Rect: rect, Z: z})
}

// AddHighlight adds a HighlightEffect.
func (dl *DisplayContext) AddHighlight(rect layout.Rectangle, style lipgloss.Style, z int) {
	dl.AddEffect(HighlightEffect{Rect: rect, Style: style, Z: z})
}

// AddPaint adds a HighlightEffect with Force enabled, overriding existing background colors.
func (dl *DisplayContext) AddPaint(rect layout.Rectangle, style lipgloss.Style, z int) {
	dl.AddEffect(HighlightEffect{Rect: rect, Style: style, Z: z, Force: true})
}

// AddInteraction adds an InteractionOp to the display context.
func (dl *DisplayContext) AddInteraction(rect layout.Rectangle, msg tea.Msg, typ InteractionType, z int) {
	dl.interactions = append(dl.interactions, interactionOp{
		InteractionOp: InteractionOp{
			Rect: rect,
			Msg:  msg,
			Type: typ,
			Z:    z,
		},
		order: dl.nextOrder(),
	})
}

// Clear removes all operations from the display context.
// Useful for reusing a DisplayContext across frames.
func (dl *DisplayContext) Clear() {
	dl.draws = dl.draws[:0]
	dl.effects = dl.effects[:0]
	dl.interactions = dl.interactions[:0]
	dl.orderCounter = 0
}

// Render executes all operations in the display context to the given screen.
// Order of execution:
// 1. Draw sorted by Z-index (low to high)
// 2. Effects sorted by Z-index (low to high)
func (dl *DisplayContext) Render(buf uv.Screen) {
	if len(dl.draws) == 0 && len(dl.effects) == 0 {
		return
	}

	ops := make([]renderOp, 0, len(dl.draws)+len(dl.effects))
	for _, op := range dl.draws {
		ops = append(ops, renderOp{
			z:      op.Z,
			order:  op.order,
			draw:   op.Draw,
			isDraw: true,
		})
	}
	for _, op := range dl.effects {
		ops = append(ops, renderOp{
			z:      op.z,
			order:  op.order,
			effect: op.effect,
		})
	}

	sort.SliceStable(ops, func(i, j int) bool {
		if ops[i].z != ops[j].z {
			return ops[i].z < ops[j].z
		}
		return ops[i].order < ops[j].order
	})

	for _, op := range ops {
		if op.isDraw {
			uv.NewStyledString(op.draw.Content).Draw(buf, op.draw.Rect)
			continue
		}
		op.effect.Apply(buf)
	}
}

// RenderToString is a convenience method that renders to a new buffer
// and returns the final string output.
func (dl *DisplayContext) RenderToString(width, height int) string {
	buf := uv.NewScreenBuffer(width, height)
	dl.Render(buf)
	return buf.Render()
}

// DrawList returns a copy of all Draw calls (useful for debugging/inspection)
func (dl *DisplayContext) DrawList() []Draw {
	result := make([]Draw, len(dl.draws))
	for i, op := range dl.draws {
		result[i] = op.Draw
	}
	return result
}

// EffectsList returns a copy of all Effects (useful for debugging/inspection)
func (dl *DisplayContext) EffectsList() []Effect {
	result := make([]Effect, len(dl.effects))
	for i, op := range dl.effects {
		result[i] = op.effect
	}
	return result
}

// InteractionsList returns all interactions sorted by Z-index (highest first for priority).
func (dl *DisplayContext) InteractionsList() []InteractionOp {
	sorted := make([]interactionOp, len(dl.interactions))
	copy(sorted, dl.interactions)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Z != sorted[j].Z {
			return sorted[i].Z > sorted[j].Z
		}
		return sorted[i].order < sorted[j].order
	})
	result := make([]InteractionOp, len(sorted))
	for i, op := range sorted {
		result[i] = op.InteractionOp
	}
	return result
}

// Merge adds all operations from another DisplayContext into this one.
func (dl *DisplayContext) Merge(other *DisplayContext) {
	for _, op := range other.draws {
		dl.draws = append(dl.draws, drawOp{
			Draw:  op.Draw,
			order: dl.nextOrder(),
		})
	}

	for _, op := range other.effects {
		dl.effects = append(dl.effects, effectOp{
			effect: op.effect,
			order:  dl.nextOrder(),
			z:      op.z,
		})
	}

	for _, op := range other.interactions {
		dl.interactions = append(dl.interactions, interactionOp{
			InteractionOp: op.InteractionOp,
			order:         dl.nextOrder(),
		})
	}
}

// Len returns the total number of operations in the display context
func (dl *DisplayContext) Len() int {
	return len(dl.draws) + len(dl.effects) + len(dl.interactions)
}

type drawOp struct {
	Draw
	order int
}

type effectOp struct {
	effect Effect
	order  int
	z      int
}

type interactionOp struct {
	InteractionOp
	order int
}

type renderOp struct {
	z      int
	order  int
	draw   Draw
	effect Effect
	isDraw bool
}

// ProcessMouseEvent routes a mouse event through the registered interactions.
func (dl *DisplayContext) ProcessMouseEvent(msg tea.MouseMsg) (tea.Msg, bool) {
	switch msg.(type) {
	case tea.MouseClickMsg, tea.MouseWheelMsg:
	default:
		return nil, false
	}

	sorted := make([]interactionOp, len(dl.interactions))
	copy(sorted, dl.interactions)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Z != sorted[j].Z {
			return sorted[i].Z > sorted[j].Z
		}
		return sorted[i].order < sorted[j].order
	})

	return processMouseEvent(sorted, msg, func(interactionOp) bool { return true })
}
