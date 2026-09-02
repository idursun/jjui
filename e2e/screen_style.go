package main

import (
	"fmt"
	"strings"

	ghostty "go.mitchellh.com/libghostty"
)

type screenCell struct {
	Text  string
	Style ghostty.RenderCellStyle
}

type styledScreen [][]screenCell

func cursorPositionSnapshot(term *ghostty.Terminal) (cursorPosition, error) {
	renderState, err := ghostty.NewRenderState()
	if err != nil {
		return cursorPosition{}, err
	}
	defer renderState.Close()

	if err := renderState.Update(term); err != nil {
		return cursorPosition{}, err
	}
	visible, err := renderState.CursorVisible()
	if err != nil {
		return cursorPosition{}, err
	}
	hasValue, err := renderState.CursorViewportHasValue()
	if err != nil {
		return cursorPosition{}, err
	}
	position := cursorPosition{Visible: visible, InViewport: hasValue}
	if !hasValue {
		return position, nil
	}
	position.X, err = renderState.CursorViewportX()
	if err != nil {
		return cursorPosition{}, err
	}
	position.Y, err = renderState.CursorViewportY()
	if err != nil {
		return cursorPosition{}, err
	}
	position.WideTail, err = renderState.CursorViewportWideTail()
	if err != nil {
		return cursorPosition{}, err
	}
	return position, nil
}

func styledScreenSnapshot(term *ghostty.Terminal) (styledScreen, error) {
	renderState, err := ghostty.NewRenderState()
	if err != nil {
		return nil, err
	}
	defer renderState.Close()

	if err := renderState.Update(term); err != nil {
		return nil, err
	}

	cols, err := renderState.Cols()
	if err != nil {
		return nil, err
	}
	rows, err := renderState.Rows()
	if err != nil {
		return nil, err
	}

	rowIterator, err := ghostty.NewRenderStateRowIterator()
	if err != nil {
		return nil, err
	}
	defer rowIterator.Close()

	cellIterator, err := ghostty.NewRenderStateRowCells()
	if err != nil {
		return nil, err
	}
	defer cellIterator.Close()

	if err := renderState.RowIterator(rowIterator); err != nil {
		return nil, err
	}

	result := make(styledScreen, 0, rows)
	for rowIterator.Next() {
		row := make([]screenCell, 0, cols)
		if err := rowIterator.Cells(cellIterator); err != nil {
			return nil, err
		}

		for cellIterator.Next() {
			text, err := cellIterator.AppendGraphemes(nil)
			if err != nil {
				return nil, err
			}
			var style ghostty.RenderCellStyle
			if err := cellIterator.StyleInto(&style); err != nil {
				return nil, err
			}
			row = append(row, screenCell{Text: string(text), Style: style})
		}

		for len(row) < int(cols) {
			row = append(row, screenCell{})
		}
		result = append(result, row)
	}

	for len(result) < int(rows) {
		result = append(result, make([]screenCell, cols))
	}
	return result, nil
}

func (s styledScreen) hasStyle(predicate func(screenCell) bool) bool {
	for _, row := range s {
		for _, cell := range row {
			if predicate(cell) {
				return true
			}
		}
	}
	return false
}

func formatStyledScreen(screen styledScreen) string {
	if len(screen) == 0 {
		return "<no screen captured>"
	}

	var result strings.Builder
	for _, row := range screen {
		for _, cell := range row {
			if cell.Text == "" {
				result.WriteByte(' ')
				continue
			}
			result.WriteString(cell.Text)
		}
		result.WriteByte('\n')
	}

	result.WriteString("\nstyled cells:\n")
	styledCount := 0
	for y, row := range screen {
		for x, cell := range row {
			style := cell.Style
			if !style.HasStyling && !style.HasForeground && !style.HasBackground {
				continue
			}
			fmt.Fprintf(&result, "(%d,%d) %q", x, y, cell.Text)
			if style.HasForeground {
				fmt.Fprintf(&result, " fg=#%02x%02x%02x", style.Foreground.R, style.Foreground.G, style.Foreground.B)
			}
			if style.HasBackground {
				fmt.Fprintf(&result, " bg=#%02x%02x%02x", style.Background.R, style.Background.G, style.Background.B)
			}
			if style.Bold {
				result.WriteString(" bold")
			}
			if style.Italic {
				result.WriteString(" italic")
			}
			if style.Underline {
				result.WriteString(" underline")
			}
			result.WriteByte('\n')
			styledCount++
			if styledCount == 32 {
				result.WriteString("...\n")
				return strings.TrimSuffix(result.String(), "\n")
			}
		}
	}
	return strings.TrimSuffix(result.String(), "\n")
}
