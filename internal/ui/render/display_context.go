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
	cursor       *tea.Cursor
	cursorPrio   int
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
func (dl *DisplayContext) AddDraw(rect layout.Rectangle, content string, z int, opts ...DrawOption) {
	var options DrawOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	dl.draws = append(dl.draws, drawOp{
		Draw: Draw{
			Rect:    rect,
			Content: content,
			Z:       z,
			Options: options,
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

// AddDim adds a DimEffect (dims the content).
func (dl *DisplayContext) AddDim(rect layout.Rectangle, z int) {
	dl.AddEffect(DimEffect{Rect: rect, Z: z})
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

// AddInteractionFn adds an interaction whose message is computed from the mouse event.
func (dl *DisplayContext) AddInteractionFn(rect layout.Rectangle, fn func(tea.MouseMsg) tea.Msg, typ InteractionType, z int) {
	dl.interactions = append(dl.interactions, interactionOp{
		InteractionOp: InteractionOp{
			Rect:  rect,
			MsgFn: fn,
			Type:  typ,
			Z:     z,
		},
		order: dl.nextOrder(),
	})
}

// SetCursor sets the real terminal cursor for the current frame.
func (dl *DisplayContext) SetCursor(cursor *tea.Cursor) {
	dl.setCursor(cursor, 0)
}

func (dl *DisplayContext) setCursor(cursor *tea.Cursor, priority int) {
	if cursor == nil {
		dl.cursor = nil
		dl.cursorPrio = 0
		return
	}
	if dl.cursor != nil && priority < dl.cursorPrio {
		return
	}
	cursorCopy := *cursor
	dl.cursor = &cursorCopy
	dl.cursorPrio = priority
}

// SetCursorAt sets the frame cursor after offsetting it to an absolute position.
func (dl *DisplayContext) SetCursorAt(cursor *tea.Cursor, x, y int) {
	if cursor == nil {
		return
	}
	cursorCopy := *cursor
	cursorCopy.Position.X += x
	cursorCopy.Position.Y += y
	dl.setCursor(&cursorCopy, 0)
}

// SetCursorInRect sets the frame cursor relative to a rectangle origin plus any local offsets.
func (dl *DisplayContext) SetCursorInRect(cursor *tea.Cursor, rect layout.Rectangle, dx, dy int) {
	dl.SetCursorAt(cursor, rect.Min.X+dx, rect.Min.Y+dy)
}

// SetInputCursorInRect sets the frame cursor for an active text editor.
// Input cursors take precedence over passive navigation cursors that may be
// rendered later in the frame.
func (dl *DisplayContext) SetInputCursorInRect(cursor *tea.Cursor, rect layout.Rectangle, dx, dy int) {
	if cursor == nil {
		return
	}
	cursorCopy := *cursor
	cursorCopy.Position.X += rect.Min.X + dx
	cursorCopy.Position.Y += rect.Min.Y + dy
	dl.setCursor(&cursorCopy, 100)
}

// Cursor returns the real terminal cursor for the current frame.
func (dl *DisplayContext) Cursor() *tea.Cursor {
	if dl.cursor == nil {
		return nil
	}
	cursorCopy := *dl.cursor
	return &cursorCopy
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
			drawStyledString(buf, op.draw)
			continue
		}
		op.effect.Apply(buf)
	}
}

func drawStyledString(buf uv.Screen, draw Draw) {
	if !draw.Options.PreserveBackground {
		uv.NewStyledString(draw.Content).Draw(buf, draw.Rect)
		return
	}

	tmp := NewScreenBuffer(draw.Rect.Dx(), draw.Rect.Dy())
	uv.NewStyledString(draw.Content).Draw(tmp, layout.Rect(0, 0, draw.Rect.Dx(), draw.Rect.Dy()))
	mergeScreenRegion(buf, tmp, draw.Rect)
}

func mergeScreenRegion(dst uv.Screen, src uv.Screen, dstRect layout.Rectangle) {
	srcBounds := src.Bounds()
	dstBounds := dst.Bounds()
	for y := 0; y < srcBounds.Dy(); y++ {
		for x := 0; x < srcBounds.Dx(); {
			srcCell := src.CellAt(x, y)
			if srcCell == nil {
				x++
				continue
			}
			if srcCell.Width == 0 {
				x++
				continue
			}

			dstX := dstRect.Min.X + x
			dstY := dstRect.Min.Y + y
			if dstX < dstBounds.Min.X || dstX >= dstBounds.Max.X || dstY < dstBounds.Min.Y || dstY >= dstBounds.Max.Y {
				if srcCell.Width > 1 {
					x += srcCell.Width
				} else {
					x++
				}
				continue
			}

			dstCell := dst.CellAt(dstX, dstY)
			merged := srcCell.Clone()
			if dstCell != nil && merged.Style.Bg == nil {
				merged.Style.Bg = dstCell.Style.Bg
			}
			dst.SetCell(dstX, dstY, merged)

			if srcCell.Width > 1 {
				x += srcCell.Width
			} else {
				x++
			}
		}
	}
}

// RenderToString is a convenience method that renders to a new buffer
// and returns the final string output.
func (dl *DisplayContext) RenderToString(width, height int) string {
	buf := NewScreenBuffer(width, height)
	dl.Render(buf)
	return buf.Render()
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
