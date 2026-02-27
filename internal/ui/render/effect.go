package render

import (
	"image/color"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/ui/layout"
)

// Effect is the interface that all effect operations must implement.
// Effects are post-processing operations that modify already-rendered content.
type Effect interface {
	// Apply applies the effect to the buffer
	Apply(buf uv.Screen)
	// GetZ returns the Z-index for layering (higher Z renders later)
	GetZ() int
	// GetRect returns the rectangle this effect applies to
	GetRect() layout.Rectangle
}

// ReverseEffect reverses foreground and background colors.
type ReverseEffect struct {
	Rect layout.Rectangle
	Z    int
}

func (e ReverseEffect) Apply(buf uv.Screen) {
	iterateCells(buf, e.Rect, func(cell *uv.Cell) *uv.Cell {
		if cell == nil {
			return nil
		}
		newCell := cell.Clone()
		newCell.Style.Attrs |= uv.AttrReverse
		return newCell
	})
}

func (e ReverseEffect) GetZ() int                 { return e.Z }
func (e ReverseEffect) GetRect() layout.Rectangle { return e.Rect }

// DimEffect dims the content by setting the Faint attribute.
type DimEffect struct {
	Rect layout.Rectangle
	Z    int
}

func (e DimEffect) Apply(buf uv.Screen) {
	iterateCells(buf, e.Rect, func(cell *uv.Cell) *uv.Cell {
		if cell == nil {
			return nil
		}
		newCell := cell.Clone()
		newCell.Style.Attrs |= uv.AttrFaint
		return newCell
	})
}

func (e DimEffect) GetZ() int                 { return e.Z }
func (e DimEffect) GetRect() layout.Rectangle { return e.Rect }

// UnderlineEffect adds underline to content.
type UnderlineEffect struct {
	Rect layout.Rectangle
	Z    int
}

func (e UnderlineEffect) Apply(buf uv.Screen) {
	iterateCells(buf, e.Rect, func(cell *uv.Cell) *uv.Cell {
		if cell == nil {
			return nil
		}
		newCell := cell.Clone()
		newCell.Style.Underline = uv.UnderlineSingle
		return newCell
	})
}

func (e UnderlineEffect) GetZ() int                 { return e.Z }
func (e UnderlineEffect) GetRect() layout.Rectangle { return e.Rect }

// BoldEffect makes content bold.
type BoldEffect struct {
	Rect layout.Rectangle
	Z    int
}

func (e BoldEffect) Apply(buf uv.Screen) {
	iterateCells(buf, e.Rect, func(cell *uv.Cell) *uv.Cell {
		if cell == nil {
			return nil
		}
		newCell := cell.Clone()
		newCell.Style.Attrs |= uv.AttrBold
		return newCell
	})
}

func (e BoldEffect) GetZ() int                 { return e.Z }
func (e BoldEffect) GetRect() layout.Rectangle { return e.Rect }

// StrikeEffect adds strikethrough to content.
type StrikeEffect struct {
	Rect layout.Rectangle
	Z    int
}

func (e StrikeEffect) Apply(buf uv.Screen) {
	iterateCells(buf, e.Rect, func(cell *uv.Cell) *uv.Cell {
		if cell == nil {
			return nil
		}
		if cell.Content != "" && cell.Content != " " {
			cell.Style.Attrs |= uv.AttrStrikethrough
		}
		return cell
	})
}

func (e StrikeEffect) GetZ() int                 { return e.Z }
func (e StrikeEffect) GetRect() layout.Rectangle { return e.Rect }

// HighlightEffect applies a highlight style by changing the background color.
// Extracts the background color from the lipgloss.Style and applies it to cells.
type HighlightEffect struct {
	Rect  layout.Rectangle
	Style lipgloss.Style
	Z     int
	Force bool
}

func (e HighlightEffect) Apply(buf uv.Screen) {
	bgColor := toAnsiColor(e.Style.GetBackground())

	iterateCells(buf, e.Rect, func(cell *uv.Cell) *uv.Cell {
		if cell == nil {
			return nil
		}
		if e.Force || cell.Style.Bg == nil {
			newCell := cell.Clone()
			newCell.Style.Bg = bgColor
			return newCell
		}
		return cell
	})
}

func (e HighlightEffect) GetZ() int                 { return e.Z }
func (e HighlightEffect) GetRect() layout.Rectangle { return e.Rect }

type FillEffect struct {
	Rect  layout.Rectangle
	Char  rune
	Style uv.Style
	Z     int
}

func (e FillEffect) Apply(buf uv.Screen) {
	cell := &uv.Cell{
		Content: string(e.Char),
		Width:   1,
		Style:   e.Style,
	}
	bounds := buf.Bounds().Intersect(e.Rect)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			buf.SetCell(x, y, cell)
		}
	}
}

func (e FillEffect) GetZ() int                 { return e.Z }
func (e FillEffect) GetRect() layout.Rectangle { return e.Rect }

// toAnsiColor converts a color.Color to the correct ansi.Color concrete type
// so that palette colors emit palette escape codes instead of 24-bit RGB.
func toAnsiColor(c color.Color) ansi.Color {
	switch c := c.(type) {
	case ansi.BasicColor:
		return c
	case ansi.IndexedColor: // = lipgloss.ANSIColor
		return c
	default:
		if ac, ok := c.(ansi.Color); ok {
			return ac
		}
		return nil
	}
}

func lipglossToStyle(ls lipgloss.Style) uv.Style {
	var cs uv.Style
	if _, isNoColor := ls.GetForeground().(lipgloss.NoColor); !isNoColor {
		cs.Fg = toAnsiColor(ls.GetForeground())
	}
	if _, isNoColor := ls.GetBackground().(lipgloss.NoColor); !isNoColor {
		cs.Bg = toAnsiColor(ls.GetBackground())
	}
	if ls.GetBold() {
		cs.Attrs |= uv.AttrBold
	}
	if ls.GetFaint() {
		cs.Attrs |= uv.AttrFaint
	}
	if ls.GetItalic() {
		cs.Attrs |= uv.AttrItalic
	}
	if ls.GetUnderline() {
		cs.Underline = uv.UnderlineSingle
	}
	if ls.GetStrikethrough() {
		cs.Attrs |= uv.AttrStrikethrough
	}
	if ls.GetReverse() {
		cs.Attrs |= uv.AttrReverse
	}
	return cs
}

// iterateCells iterates over all cells in a rectangle, applies a transformation,
// and writes the modified cells back to the buffer.
func iterateCells(buf uv.Screen, rect layout.Rectangle, transform func(*uv.Cell) *uv.Cell) {
	bounds := buf.Bounds()
	// Clamp rect to buffer bounds
	rect = rect.Intersect(bounds)

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; {
			cell := buf.CellAt(x, y)
			if cell == nil {
				x++
				continue
			}

			// Skip width-0 placeholder cells that belong to wide graphemes.
			// Writing to these positions causes the buffer to blank the leading cell,
			// which corrupts emoji/CJK rendering when applying effects.
			if cell.Width == 0 {
				x++
				continue
			}

			newCell := transform(cell)
			if newCell != nil {
				buf.SetCell(x, y, newCell)
			}

			if cell.Width > 1 {
				x += cell.Width
			} else {
				x++
			}
		}
	}
}
