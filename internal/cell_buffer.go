package ui

import (
	"fmt"
	"log"
	"strings"

	"github.com/leaanthony/go-ansi-parser"
)

type cell struct {
	char  rune
	fg    *ansi.Col
	bg    *ansi.Col
	style ansi.TextStyle
}

type cellBuffer struct {
	grid [][]cell
}

func Stacked(view1, view2 string, x, y int) (string, error) {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	buf := &cellBuffer{}

	// Parse and apply base view
	if err := buf.applyANSI(view1, 0, 0); err != nil {
		return "", err
	}

	// Parse and apply overlay view
	if err := buf.applyANSI(view2, x, y); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (b *cellBuffer) applyANSI(input string, offsetX, offsetY int) error {
	parsed, err := ansi.Parse(input)
	if err != nil {
		return err
	}

	currentLine := offsetY
	currentCol := offsetX
	for _, st := range parsed {
		for _, char := range st.Label {
			if char == '\n' {
				currentLine++
				currentCol = offsetX
				continue
			}

			// Expand buffer as needed
			for currentLine >= len(b.grid) {
				b.grid = append(b.grid, []cell{})
			}
			for currentCol >= len(b.grid[currentLine]) {
				b.grid[currentLine] = append(b.grid[currentLine], cell{char: ' '})
			}

			// Overwrite cell
			if currentCol < 0 || currentLine < 0 {
				log.Fatalf("line: %d, col: %d", currentLine, currentCol)
			}
			b.grid[currentLine][currentCol] = cell{
				char:  char,
				fg:    st.FgCol,
				bg:    st.BgCol,
				style: st.Style,
			}
			currentCol++
		}
	}
	return nil
}

func (b *cellBuffer) String() string {
	var sb strings.Builder
	for lineNum, line := range b.grid {
		if lineNum > 0 {
			sb.WriteByte('\n')
		}

		var currentFG, currentBG *ansi.Col
		var currentStyle ansi.TextStyle

		for _, c := range line {
			if c.fg != currentFG || c.bg != currentBG || c.style != currentStyle {
				writeStyle(&sb, c.fg, c.bg, c.style)
				currentFG = c.fg
				currentBG = c.bg
				currentStyle = c.style
			}
			sb.WriteRune(c.char)
		}

		// Reset styles at end of line
		if currentFG != nil || currentBG != nil || currentStyle != 0 {
			sb.WriteString("\x1b[0m")
		}
	}
	return sb.String()
}

func writeStyle(sb *strings.Builder, fg, bg *ansi.Col, style ansi.TextStyle) {
	var codes []string

	// Handle text styles
	if style&ansi.Bold != 0 {
		codes = append(codes, "1")
	}
	if style&ansi.Italic != 0 {
		codes = append(codes, "3")
	}
	if style&ansi.Underlined != 0 {
		codes = append(codes, "4")
	}

	// Handle foreground color
	if fg != nil {
		if fg.Id < 16 {
			codes = append(codes, fgAnsiCode(fg.Id))
		} else if fg.Id < 256 {
			codes = append(codes, fg256Code(fg.Id))
		} else {
			codes = append(codes, fgRgbCode(fg.Rgb.R, fg.Rgb.G, fg.Rgb.B))
		}
	}

	// Handle background color
	if bg != nil {
		if bg.Id < 16 {
			codes = append(codes, bgAnsiCode(bg.Id))
		} else if bg.Id < 256 {
			codes = append(codes, bg256Code(bg.Id))
		} else {
			codes = append(codes, bgRgbCode(bg.Rgb.R, bg.Rgb.G, bg.Rgb.B))
		}
	}

	// Write ANSI escape sequence
	if len(codes) > 0 {
		sb.WriteString("\x1b[")
		sb.WriteString(strings.Join(codes, ";"))
		sb.WriteString("m")
	} else {
		sb.WriteString("\x1b[0m")
	}
}

// Helper functions for ANSI code generation
func fgAnsiCode(id int) string       { return fmt.Sprintf("%d", 30+(id%8)) }
func bgAnsiCode(id int) string       { return fmt.Sprintf("%d", 40+(id%8)) }
func fg256Code(id int) string        { return fmt.Sprintf("38;5;%d", id) }
func bg256Code(id int) string        { return fmt.Sprintf("48;5;%d", id) }
func fgRgbCode(r, g, b uint8) string { return fmt.Sprintf("38;2;%d;%d;%d", r, g, b) }
func bgRgbCode(r, g, b uint8) string { return fmt.Sprintf("48;2;%d;%d;%d", r, g, b) }
