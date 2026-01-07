package ops

import (
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
)

// DrawOp represents a content rendering operation.
// DrawOps are rendered first, sorted by Z-index (lower values render first).
type DrawOp struct {
	Rect    cellbuf.Rectangle // The area to draw in
	Content string            // Rendered ANSI string (from lipgloss, etc.)
	Z       int               // Z-index for layering (lower = back, higher = front)
}

// Effect is the interface that all effect operations must implement.
// Effects are post-processing operations that modify already-rendered content.
type Effect interface {
	// Apply applies the effect to the buffer
	Apply(buf *cellbuf.Buffer)
	// GetZ returns the Z-index for layering (higher Z renders later)
	GetZ() int
	// GetRect returns the rectangle this effect applies to
	GetRect() cellbuf.Rectangle
}

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
	ID   string            // Identifier for this interaction (sent in messages)
	Type InteractionType   // What kind of interaction this supports
	Z    int               // Z-index for overlapping regions (higher = priority)
}

// ReverseEffect reverses foreground and background colors.
type ReverseEffect struct {
	Rect cellbuf.Rectangle
	Z    int
}

func (e ReverseEffect) Apply(buf *cellbuf.Buffer) {
	iterateCells(buf, e.Rect, func(cell *cellbuf.Cell) *cellbuf.Cell {
		if cell == nil {
			return nil
		}
		newCell := cell.Clone()
		newCell.Style.Reverse(true)
		return newCell
	})
}

func (e ReverseEffect) GetZ() int                  { return e.Z }
func (e ReverseEffect) GetRect() cellbuf.Rectangle { return e.Rect }

// DimEffect dims the content by setting the Faint attribute.
type DimEffect struct {
	Rect cellbuf.Rectangle
	Z    int
}

func (e DimEffect) Apply(buf *cellbuf.Buffer) {
	iterateCells(buf, e.Rect, func(cell *cellbuf.Cell) *cellbuf.Cell {
		if cell == nil {
			return nil
		}
		newCell := cell.Clone()
		newCell.Style.Faint(true)
		return newCell
	})
}

func (e DimEffect) GetZ() int                  { return e.Z }
func (e DimEffect) GetRect() cellbuf.Rectangle { return e.Rect }

// UnderlineEffect adds underline to content.
type UnderlineEffect struct {
	Rect cellbuf.Rectangle
	Z    int
}

func (e UnderlineEffect) Apply(buf *cellbuf.Buffer) {
	iterateCells(buf, e.Rect, func(cell *cellbuf.Cell) *cellbuf.Cell {
		if cell == nil {
			return nil
		}
		newCell := cell.Clone()
		newCell.Style.Underline(true)
		return newCell
	})
}

func (e UnderlineEffect) GetZ() int                  { return e.Z }
func (e UnderlineEffect) GetRect() cellbuf.Rectangle { return e.Rect }

// BoldEffect makes content bold.
type BoldEffect struct {
	Rect cellbuf.Rectangle
	Z    int
}

func (e BoldEffect) Apply(buf *cellbuf.Buffer) {
	iterateCells(buf, e.Rect, func(cell *cellbuf.Cell) *cellbuf.Cell {
		if cell == nil {
			return nil
		}
		newCell := cell.Clone()
		newCell.Style.Bold(true)
		return newCell
	})
}

func (e BoldEffect) GetZ() int                  { return e.Z }
func (e BoldEffect) GetRect() cellbuf.Rectangle { return e.Rect }

// StrikeEffect adds strikethrough to content.
type StrikeEffect struct {
	Rect cellbuf.Rectangle
	Z    int
}

func (e StrikeEffect) Apply(buf *cellbuf.Buffer) {
	iterateCells(buf, e.Rect, func(cell *cellbuf.Cell) *cellbuf.Cell {
		if cell == nil {
			return nil
		}
		if cell.Rune != 0 && cell.Rune != ' ' {
			cell.Style.Strikethrough(true)
		}
		return cell
	})
}

func (e StrikeEffect) GetZ() int                  { return e.Z }
func (e StrikeEffect) GetRect() cellbuf.Rectangle { return e.Rect }

// HighlightEffect applies a highlight style by changing the background color.
// Extracts the background color from the lipgloss.Style and applies it to cells.
type HighlightEffect struct {
	Rect  cellbuf.Rectangle
	Style lipgloss.Style
	Z     int
}

func (e HighlightEffect) Apply(buf *cellbuf.Buffer) {
	// Extract background color from lipgloss.Style
	bgColor := e.Style.GetBackground()

	iterateCells(buf, e.Rect, func(cell *cellbuf.Cell) *cellbuf.Cell {
		if cell == nil {
			return nil
		}
		// Apply the background color from the style
		if cell.Style.Bg == nil {
			cell.Style.Background(bgColor)
		}
		return cell
	})
}

func (e HighlightEffect) GetZ() int                  { return e.Z }
func (e HighlightEffect) GetRect() cellbuf.Rectangle { return e.Rect }

// DisplayList holds all rendering operations for a frame.
// Operations are accumulated during the layout/render pass,
// then executed in order (DrawOps by Z, then Effects by Z).
type DisplayList struct {
	draws        []DrawOp
	effects      []Effect
	interactions []InteractionOp
}

// NewDisplayList creates a new empty display list.
func NewDisplayList() *DisplayList {
	return &DisplayList{
		draws:        make([]DrawOp, 0, 16),
		effects:      make([]Effect, 0, 8),
		interactions: make([]InteractionOp, 0, 8),
	}
}

// FromString creates a DisplayList from a simple string view (no interactions).
// Useful for models that don't need mouse interaction.
func FromString(content string, rect cellbuf.Rectangle) *DisplayList {
	dl := NewDisplayList()
	dl.AddDraw(rect, content, 0)
	return dl
}

// AddDraw adds a DrawOp to the display list.
func (dl *DisplayList) AddDraw(rect cellbuf.Rectangle, content string, z int) {
	dl.draws = append(dl.draws, DrawOp{
		Rect:    rect,
		Content: content,
		Z:       z,
	})
}

// AddEffectOp adds a custom Effect to the display list.
// This is the generic method that accepts any Effect implementation.
func (dl *DisplayList) AddEffectOp(effect Effect) {
	dl.effects = append(dl.effects, effect)
}

// AddReverse adds a ReverseEffect (reverses foreground/background colors).
func (dl *DisplayList) AddReverse(rect cellbuf.Rectangle, z int) {
	dl.AddEffectOp(ReverseEffect{Rect: rect, Z: z})
}

// AddDim adds a DimEffect (dims the content).
func (dl *DisplayList) AddDim(rect cellbuf.Rectangle, z int) {
	dl.AddEffectOp(DimEffect{Rect: rect, Z: z})
}

// AddUnderline adds an UnderlineEffect.
func (dl *DisplayList) AddUnderline(rect cellbuf.Rectangle, z int) {
	dl.AddEffectOp(UnderlineEffect{Rect: rect, Z: z})
}

// AddBold adds a BoldEffect.
func (dl *DisplayList) AddBold(rect cellbuf.Rectangle, z int) {
	dl.AddEffectOp(BoldEffect{Rect: rect, Z: z})
}

// AddStrike adds a StrikeEffect (strikethrough).
func (dl *DisplayList) AddStrike(rect cellbuf.Rectangle, z int) {
	dl.AddEffectOp(StrikeEffect{Rect: rect, Z: z})
}

// AddHighlight adds a HighlightEffect.
func (dl *DisplayList) AddHighlight(rect cellbuf.Rectangle, style lipgloss.Style, z int) {
	dl.AddEffectOp(HighlightEffect{Rect: rect, Style: style, Z: z})
}

// AddInteraction adds an InteractionOp to the display list.
func (dl *DisplayList) AddInteraction(rect cellbuf.Rectangle, id string, typ InteractionType, z int) {
	dl.interactions = append(dl.interactions, InteractionOp{
		Rect: rect,
		ID:   id,
		Type: typ,
		Z:    z,
	})
}

// AddClickable adds a clickable interactive region.
func (dl *DisplayList) AddClickable(rect cellbuf.Rectangle, id string, z int) {
	dl.AddInteraction(rect, id, InteractionClick, z)
}

// AddScrollable adds a scrollable interactive region.
func (dl *DisplayList) AddScrollable(rect cellbuf.Rectangle, id string, z int) {
	dl.AddInteraction(rect, id, InteractionScroll, z)
}

// AddDraggable adds a draggable interactive region.
func (dl *DisplayList) AddDraggable(rect cellbuf.Rectangle, id string, z int) {
	dl.AddInteraction(rect, id, InteractionDrag, z)
}

// Deprecated: AddEffect is the old API. Use specific helpers (AddReverse, AddDim, etc.) instead.
// Kept for backward compatibility with existing code.
func (dl *DisplayList) AddEffect(rect cellbuf.Rectangle, mode EffectMode, style lipgloss.Style, z int) {
	switch mode {
	case ModeReverse:
		dl.AddReverse(rect, z)
	case ModeDim:
		dl.AddDim(rect, z)
	case ModeUnderline:
		dl.AddUnderline(rect, z)
	case ModeBold:
		dl.AddBold(rect, z)
	case ModeHighlight:
		dl.AddHighlight(rect, style, z)
	}
}

// Deprecated: EffectMode is the old API. Use specific effect types instead.
type EffectMode int

const (
	ModeDim EffectMode = iota
	ModeReverse
	ModeHighlight
	ModeUnderline
	ModeBold
)

// Clear removes all operations from the display list.
// Useful for reusing a DisplayList across frames.
func (dl *DisplayList) Clear() {
	dl.draws = dl.draws[:0]
	dl.effects = dl.effects[:0]
	dl.interactions = dl.interactions[:0]
}

// Render executes all operations in the display list to the given cellbuf.
// Order of execution:
// 1. DrawOps sorted by Z-index (low to high)
// 2. Effects sorted by Z-index (low to high)
func (dl *DisplayList) Render(buf *cellbuf.Buffer) {
	// Sort DrawOps by Z-index (stable sort maintains insertion order for equal Z)
	sort.SliceStable(dl.draws, func(i, j int) bool {
		return dl.draws[i].Z < dl.draws[j].Z
	})

	// Render all DrawOps
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

// iterateCells iterates over all cells in a rectangle, applies a transformation,
// and writes the modified cells back to the buffer.
func iterateCells(buf *cellbuf.Buffer, rect cellbuf.Rectangle, transform func(*cellbuf.Cell) *cellbuf.Cell) {
	bounds := buf.Bounds()
	// Clamp rect to buffer bounds
	rect = rect.Intersect(bounds)

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			cell := buf.Cell(x, y)
			newCell := transform(cell)
			if newCell != nil {
				buf.SetCell(x, y, newCell)
			}
		}
	}
}

// DrawOpsList returns a copy of all DrawOps (useful for debugging/inspection)
func (dl *DisplayList) DrawOpsList() []DrawOp {
	result := make([]DrawOp, len(dl.draws))
	copy(result, dl.draws)
	return result
}

// EffectsList returns a copy of all Effects (useful for debugging/inspection)
func (dl *DisplayList) EffectsList() []Effect {
	result := make([]Effect, len(dl.effects))
	copy(result, dl.effects)
	return result
}

// Deprecated: EffectOpsList is the old API. Use EffectsList() instead.
func (dl *DisplayList) EffectOpsList() []EffectOp {
	// Convert Effects back to old EffectOp format for compatibility
	result := make([]EffectOp, 0, len(dl.effects))
	for _, effect := range dl.effects {
		mode := ModeDim // default
		var style lipgloss.Style

		switch e := effect.(type) {
		case ReverseEffect:
			mode = ModeReverse
		case DimEffect:
			mode = ModeDim
		case UnderlineEffect:
			mode = ModeUnderline
		case BoldEffect:
			mode = ModeBold
		case HighlightEffect:
			mode = ModeHighlight
			style = e.Style
		}

		result = append(result, EffectOp{
			Rect:  effect.GetRect(),
			Mode:  mode,
			Style: style,
			Z:     effect.GetZ(),
		})
	}
	return result
}

// Deprecated: EffectOp is the old API. Use specific effect types instead.
type EffectOp struct {
	Rect  cellbuf.Rectangle
	Mode  EffectMode
	Style lipgloss.Style
	Z     int
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
