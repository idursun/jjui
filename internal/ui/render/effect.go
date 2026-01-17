package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
)

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

func fillString(width, height int, ch rune, style lipgloss.Style) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	line := strings.Repeat(string(ch), width)
	styledLine := style.Render(line)
	var b strings.Builder
	b.Grow((len(styledLine) + 1) * height)
	for i := 0; i < height; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(styledLine)
	}
	return b.String()
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
