package render

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
)

// DisplayContext holds all rendering operations for a frame.
// Operations are accumulated during the layout/render pass,
// then executed in order by batch and Z-index.
type DisplayContext struct {
	draws         []drawOp
	effects       []effectOp
	interactions  []interactionOp
	orderCounter  int
	windows       []windowOp
	windowCounter int
	parent        *DisplayContext
	windowID      int
}

// NewDisplayContext creates a new empty display context.
func NewDisplayContext() *DisplayContext {
	return &DisplayContext{
		draws:        make([]drawOp, 0, 16),
		effects:      make([]effectOp, 0, 8),
		interactions: make([]interactionOp, 0, 8),
		windows:      make([]windowOp, 0, 4),
	}
}

// Window creates a scoped display context that routes interactions to a window.
func (dl *DisplayContext) Window(rect cellbuf.Rectangle, z int) *DisplayContext {
	root := dl.root()
	root.windowCounter++
	id := root.windowCounter
	root.windows = append(root.windows, windowOp{
		ID:    id,
		Rect:  rect,
		Z:     z,
		Order: root.nextOrder(),
	})
	return &DisplayContext{parent: root, windowID: id}
}

func (dl *DisplayContext) root() *DisplayContext {
	if dl.parent == nil {
		return dl
	}
	return dl.parent
}

func (dl *DisplayContext) nextOrder() int {
	root := dl.root()
	root.orderCounter++
	return root.orderCounter
}

func (dl *DisplayContext) currentWindowID() int {
	if dl.parent == nil {
		return 0
	}
	return dl.windowID
}

// AddDraw adds a Draw to the display context.
func (dl *DisplayContext) AddDraw(rect cellbuf.Rectangle, content string, z int) {
	root := dl.root()
	root.draws = append(root.draws, drawOp{
		Draw: Draw{
			Rect:    rect,
			Content: content,
			Z:       z,
		},
		order: dl.nextOrder(),
	})
}

// AddFill fills a rectangle with the provided rune and style.
func (dl *DisplayContext) AddFill(rect cellbuf.Rectangle, ch rune, style lipgloss.Style, z int) {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	content := fillString(rect.Dx(), rect.Dy(), ch, style)
	if content == "" {
		return
	}
	dl.AddDraw(rect, content, z)
}

// AddEffect adds a custom Effect to the display context.
// This is the generic method that accepts any Effect implementation.
func (dl *DisplayContext) AddEffect(effect Effect) {
	root := dl.root()
	root.effects = append(root.effects, effectOp{
		effect: effect,
		order:  dl.nextOrder(),
		z:      effect.GetZ(),
	})
}

// AddReverse adds a ReverseEffect (reverses foreground/background colors).
func (dl *DisplayContext) AddReverse(rect cellbuf.Rectangle, z int) {
	dl.AddEffect(ReverseEffect{Rect: rect, Z: z})
}

// AddDim adds a DimEffect (dims the content).
func (dl *DisplayContext) AddDim(rect cellbuf.Rectangle, z int) {
	dl.AddEffect(DimEffect{Rect: rect, Z: z})
}

// AddUnderline adds an UnderlineEffect.
func (dl *DisplayContext) AddUnderline(rect cellbuf.Rectangle, z int) {
	dl.AddEffect(UnderlineEffect{Rect: rect, Z: z})
}

// AddBold adds a BoldEffect.
func (dl *DisplayContext) AddBold(rect cellbuf.Rectangle, z int) {
	dl.AddEffect(BoldEffect{Rect: rect, Z: z})
}

// AddStrike adds a StrikeEffect (strikethrough).
func (dl *DisplayContext) AddStrike(rect cellbuf.Rectangle, z int) {
	dl.AddEffect(StrikeEffect{Rect: rect, Z: z})
}

// AddHighlight adds a HighlightEffect.
func (dl *DisplayContext) AddHighlight(rect cellbuf.Rectangle, style lipgloss.Style, z int) {
	dl.AddEffect(HighlightEffect{Rect: rect, Style: style, Z: z})
}

// AddInteraction adds an InteractionOp to the display context.
func (dl *DisplayContext) AddInteraction(rect cellbuf.Rectangle, msg tea.Msg, typ InteractionType, z int) {
	root := dl.root()
	root.interactions = append(root.interactions, interactionOp{
		InteractionOp: InteractionOp{
			Rect: rect,
			Msg:  msg,
			Type: typ,
			Z:    z,
		},
		windowID: dl.currentWindowID(),
		order:    dl.nextOrder(),
	})
}

// Clear removes all operations from the display context.
// Useful for reusing a DisplayContext across frames.
func (dl *DisplayContext) Clear() {
	root := dl.root()
	root.draws = root.draws[:0]
	root.effects = root.effects[:0]
	root.interactions = root.interactions[:0]
	root.windows = root.windows[:0]
	root.orderCounter = 0
	root.windowCounter = 0
}

// Render executes all operations in the display context to the given cellbuf.
// Order of execution:
// 1. Draw sorted by Z-index (low to high)
// 2. Effects sorted by Z-index (low to high)
func (dl *DisplayContext) Render(buf *cellbuf.Buffer) {
	root := dl.root()
	if root != dl {
		root.Render(buf)
		return
	}

	if len(root.draws) == 0 && len(root.effects) == 0 {
		return
	}

	ops := make([]renderOp, 0, len(root.draws)+len(root.effects))
	for _, op := range root.draws {
		ops = append(ops, renderOp{
			z:      op.Z,
			order:  op.order,
			draw:   op.Draw,
			isDraw: true,
		})
	}
	for _, op := range root.effects {
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
			cellbuf.SetContentRect(buf, op.draw.Content, op.draw.Rect)
			continue
		}
		op.effect.Apply(buf)
	}
}

// RenderToString is a convenience method that renders to a new buffer
// and returns the final string output.
func (dl *DisplayContext) RenderToString(width, height int) string {
	buf := cellbuf.NewBuffer(width, height)
	dl.Render(buf)
	return cellbuf.Render(buf)
}

// DrawList returns a copy of all Draw calls (useful for debugging/inspection)
func (dl *DisplayContext) DrawList() []Draw {
	root := dl.root()
	result := make([]Draw, len(root.draws))
	for i, op := range root.draws {
		result[i] = op.Draw
	}
	return result
}

// EffectsList returns a copy of all Effects (useful for debugging/inspection)
func (dl *DisplayContext) EffectsList() []Effect {
	root := dl.root()
	result := make([]Effect, len(root.effects))
	for i, op := range root.effects {
		result[i] = op.effect
	}
	return result
}

// InteractionsList returns all interactions sorted by Z-index (highest first for priority).
func (dl *DisplayContext) InteractionsList() []InteractionOp {
	root := dl.root()
	sorted := make([]interactionOp, len(root.interactions))
	copy(sorted, root.interactions)
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
	root := dl.root()
	source := other.root()

	windowMap := make(map[int]int, len(source.windows))
	for _, win := range source.windows {
		root.windowCounter++
		newID := root.windowCounter
		windowMap[win.ID] = newID
		root.windows = append(root.windows, windowOp{
			ID:    newID,
			Rect:  win.Rect,
			Z:     win.Z,
			Order: root.nextOrder(),
		})
	}

	for _, op := range source.draws {
		root.draws = append(root.draws, drawOp{
			Draw:  op.Draw,
			order: root.nextOrder(),
		})
	}

	for _, op := range source.effects {
		root.effects = append(root.effects, effectOp{
			effect: op.effect,
			order:  root.nextOrder(),
			z:      op.z,
		})
	}

	for _, op := range source.interactions {
		windowID := op.windowID
		if windowID != 0 {
			if remapped, ok := windowMap[windowID]; ok {
				windowID = remapped
			}
		}
		root.interactions = append(root.interactions, interactionOp{
			InteractionOp: op.InteractionOp,
			windowID:      windowID,
			order:         root.nextOrder(),
		})
	}
}

// Len returns the total number of operations in the display context
func (dl *DisplayContext) Len() int {
	root := dl.root()
	return len(root.draws) + len(root.effects) + len(root.interactions)
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
	windowID int
	order    int
}

type renderOp struct {
	z      int
	order  int
	draw   Draw
	effect Effect
	isDraw bool
}

type windowOp struct {
	ID    int
	Rect  cellbuf.Rectangle
	Z     int
	Order int
}

// ProcessMouseEvent routes a mouse event through the window stack.
func (dl *DisplayContext) ProcessMouseEvent(msg tea.MouseMsg) (tea.Msg, bool) {
	root := dl.root()
	return ProcessMouseEventWithWindows(root.interactions, root.windows, msg)
}
