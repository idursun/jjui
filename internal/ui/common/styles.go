package common

import (
	"github.com/idursun/jjui/internal/config"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	Black         = lipgloss.Color("0")
	Red           = lipgloss.Color("1")
	Green         = lipgloss.Color("2")
	Yellow        = lipgloss.Color("3")
	Blue          = lipgloss.Color("4")
	Magenta       = lipgloss.Color("5")
	Cyan          = lipgloss.Color("6")
	White         = lipgloss.Color("7")
	BrightBlack   = lipgloss.Color("8")
	BrightRed     = lipgloss.Color("9")
	BrightGreen   = lipgloss.Color("10")
	BrightYellow  = lipgloss.Color("11")
	BrightBlue    = lipgloss.Color("12")
	BrightMagenta = lipgloss.Color("13")
	BrightCyan    = lipgloss.Color("14")
	BrightWhite   = lipgloss.Color("15")
)

var DefaultPalette = Palette{
	Normal: lipgloss.NewStyle(),
}

type Palette struct {
	Normal lipgloss.Style
	styles map[string]lipgloss.Style
}

func (p *Palette) Update(styleMap map[string]config.Color) {
	if p.styles == nil {
		p.styles = make(map[string]lipgloss.Style)
	}

	for key, color := range styleMap {
		p.styles[key] = createStyleFrom(color)
	}

	if color, ok := styleMap["diff added"]; ok {
		p.styles["details added"] = createStyleFrom(color)
	}
	if color, ok := styleMap["diff renamed"]; ok {
		p.styles["details renamed"] = createStyleFrom(color)
	}
	if color, ok := styleMap["diff modified"]; ok {
		p.styles["details modified"] = createStyleFrom(color)
	}
	if color, ok := styleMap["diff removed"]; ok {
		p.styles["details deleted"] = createStyleFrom(color)
	}
}

func (p *Palette) Get(selector string) lipgloss.Style {
	if style, ok := p.styles[selector]; ok {
		return style
	}

	fields := strings.Fields(selector)
	finalStyle := lipgloss.NewStyle()
	for _, field := range fields {
		if style, ok := p.styles[field]; ok {
			finalStyle = finalStyle.Inherit(style)
		}
	}
	return finalStyle
}

func createStyleFrom(color config.Color) lipgloss.Style {
	style := lipgloss.NewStyle()
	if color.Fg != "" {
		style = style.Foreground(parseColor(color.Fg))
	}
	if color.Bg != "" {
		style = style.Background(parseColor(color.Bg))
	}
	if color.Bold {
		style = style.Bold(true)
	}
	if color.Underline {
		style = style.Underline(true)
	}
	return style
}

func parseColor(color string) lipgloss.Color {
	// if it's a hex color, return it directly
	if len(color) == 7 && color[0] == '#' {
		return lipgloss.Color(color)
	}
	// if it's an ANSI256 color, return it directly
	if v, err := strconv.Atoi(color); err == nil {
		if v >= 0 && v <= 255 {
			return lipgloss.Color(color)
		}
	}
	// otherwise, try to parse it as a named color
	switch color {
	case "black":
		return "0"
	case "red":
		return "1"
	case "green":
		return "2"
	case "yellow":
		return "3"
	case "blue":
		return "4"
	case "magenta":
		return "5"
	case "cyan":
		return "6"
	case "white":
		return "7"
	case "bright black":
		return "8"
	case "bright red":
		return "9"
	case "bright green":
		return "10"
	case "bright yellow":
		return "11"
	case "bright blue":
		return "12"
	case "bright magenta":
		return "13"
	case "bright cyan":
		return "14"
	case "bright white":
		return "15"
	default:
		if strings.HasPrefix(color, "ansi-color-") {
			code := strings.TrimPrefix(color, "ansi-color-")
			if v, err := strconv.Atoi(code); err == nil && v >= 0 && v <= 255 {
				return lipgloss.Color(code)
			}
		}
		return ""
	}
}
