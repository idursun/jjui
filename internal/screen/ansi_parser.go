package screen

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

func applyParamsToStyle(style lipgloss.Style, param string) lipgloss.Style {
	if param == "" {
		return style
	}

	parts := strings.Split(param, ";")

	for i := 0; i < len(parts); i++ {
		code, err := strconv.Atoi(parts[i])
		if err != nil {
			continue
		}

		switch {
		case code == 0:
			// Reset all
			style = lipgloss.NewStyle()
		case code == 1:
			// Bold
			style = style.Bold(true)
		case code == 2:
			// Dim
			style = style.Faint(true)
		case code == 3:
			// Italic
			style = style.Italic(true)
		case code == 4:
			// Underline
			style = style.Underline(true)
		case code == 5 || code == 6:
			// Blink (slow or rapid)
			style = style.Blink(true)
		case code == 7:
			// Reverse
			style = style.Reverse(true)
		case code == 8:
			// Hidden
			// Not directly supported in lipgloss, could use faint + background
			style = style.Faint(true)
		case code == 9:
			// Strikethrough
			style = style.Strikethrough(true)
		case code == 22:
			// Disable bold/dim
			style = style.UnsetBold().UnsetFaint()
		case code == 23:
			// Disable italic
			style = style.UnsetItalic()
		case code == 24:
			// Disable underline
			style = style.UnsetUnderline()
		case code == 25:
			// Disable blink
			style = style.UnsetBlink()
		case code == 27:
			// Disable reverse
			style = style.UnsetReverse()
		case code == 28:
			// Disable hidden (not directly supported, but we can try to reset faint)
			style = style.UnsetFaint()
		case code == 29:
			// Disable strikethrough
			style = style.UnsetStrikethrough()
		case code >= 30 && code <= 37:
			// Foreground color
			style = style.Foreground(lipgloss.ANSIColor(code - 30))
		case code == 38 && i+2 < len(parts):
			// Extended foreground color
			if parts[i+1] == "5" && i+2 < len(parts) { // 8-bit color
				colorIndex, err := strconv.Atoi(parts[i+2])
				if err == nil {
					style = style.Foreground(lipgloss.Color(strconv.Itoa(colorIndex)))
				}
				i += 2
			} else if parts[i+1] == "2" && i+4 < len(parts) { // 24-bit color
				r, errR := strconv.Atoi(parts[i+2])
				g, errG := strconv.Atoi(parts[i+3])
				b, errB := strconv.Atoi(parts[i+4])
				if errR == nil && errG == nil && errB == nil {
					hexColor := fmt.Sprintf("#%02x%02x%02x", r, g, b)
					style = style.Foreground(lipgloss.Color(hexColor))
				}
				i += 4
			}
		case code == 39:
			// Default foreground
			style = style.UnsetForeground()
		case code >= 40 && code <= 47:
			// Background color
			style = style.Background(lipgloss.ANSIColor(code - 40))
		case code == 48 && i+2 < len(parts):
			// Extended background color
			if parts[i+1] == "5" && i+2 < len(parts) { // 8-bit color
				colorIndex, err := strconv.Atoi(parts[i+2])
				if err == nil {
					style = style.Background(lipgloss.Color(strconv.Itoa(colorIndex)))
				}
				i += 2
			} else if parts[i+1] == "2" && i+4 < len(parts) { // 24-bit color
				r, errR := strconv.Atoi(parts[i+2])
				g, errG := strconv.Atoi(parts[i+3])
				b, errB := strconv.Atoi(parts[i+4])
				if errR == nil && errG == nil && errB == nil {
					hexColor := fmt.Sprintf("#%02x%02x%02x", r, g, b)
					style = style.Background(lipgloss.Color(hexColor))
				}
				i += 4
			}
		case code == 49:
			// Default background
			style = style.UnsetBackground()
		case code >= 90 && code <= 97:
			// Bright foreground color
			style = style.Foreground(lipgloss.ANSIColor(code - 90 + 8))
		case code >= 100 && code <= 107:
			// Bright background color
			style = style.Background(lipgloss.ANSIColor(code - 100 + 8))
		}
	}

	return style
}
