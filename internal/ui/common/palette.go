package common

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/idursun/jjui/internal/config"

	"charm.land/lipgloss/v2"
)

var DefaultPalette = NewPalette()

type node struct {
	style    lipgloss.Style
	children map[string]*node
}

type Palette struct {
	root  *node
	cache map[string]lipgloss.Style
}

func NewPalette() *Palette {
	return &Palette{
		root:  nil,
		cache: make(map[string]lipgloss.Style),
	}
}

func (p *Palette) add(key string, style lipgloss.Style) {
	if p.root == nil {
		p.root = &node{children: make(map[string]*node)}
	}
	current := p.root
	prefixes := strings.Fields(key)
	for _, prefix := range prefixes {
		if child, ok := current.children[prefix]; ok {
			current = child
		} else {
			child = &node{children: make(map[string]*node)}
			current.children[prefix] = child
			current = child
		}
	}
	current.style = style
}

func (p *Palette) get(fields ...string) lipgloss.Style {
	if p.root == nil {
		return lipgloss.NewStyle()
	}

	current := p.root
	for _, field := range fields {
		if child, ok := current.children[field]; ok {
			current = child
		} else {
			return lipgloss.NewStyle() // Return default style if not found
		}
	}

	return current.style
}

func (p *Palette) Update(styleMap map[string]config.Color) {
	for key, color := range styleMap {
		p.add(key, createStyleFrom(color))
	}

	if color, ok := styleMap["diff added"]; ok {
		p.add("added", createStyleFrom(color))
	}
	if color, ok := styleMap["diff renamed"]; ok {
		p.add("renamed", createStyleFrom(color))
	}
	if color, ok := styleMap["diff copied"]; ok {
		p.add("copied", createStyleFrom(color))
	}
	if color, ok := styleMap["diff modified"]; ok {
		p.add("modified", createStyleFrom(color))
	}
	if color, ok := styleMap["diff removed"]; ok {
		p.add("deleted", createStyleFrom(color))
	}
}

func (p *Palette) Get(selector string) lipgloss.Style {
	if style, ok := p.cache[selector]; ok {
		return style
	}
	fields := strings.Fields(selector)
	length := len(fields)

	finalStyle := lipgloss.NewStyle()
	// for a selector like "a b c", we want to inherit styles from the most specific to the least specific
	// first pass: "a b c", "a b", "a"
	// second pass: "b c", "b"
	// third pass: "c"
	start := 0
	for start < length {
		for end := length; end > start; end-- {
			finalStyle = finalStyle.Inherit(p.get(fields[start:end]...))
		}
		start++
	}
	p.cache[selector] = finalStyle
	return finalStyle
}

func (p *Palette) GetBorder(selector string, border lipgloss.Border) lipgloss.Style {
	style := p.Get(selector)
	return lipgloss.NewStyle().
		Border(border).
		Foreground(style.GetForeground()).
		Background(style.GetBackground()).
		BorderForeground(style.GetForeground()).
		BorderBackground(style.GetBackground())
}

func createStyleFrom(color config.Color) lipgloss.Style {
	style := lipgloss.NewStyle()
	if color.Fg != "" {
		style = style.Foreground(parseColor(color.Fg))
	}
	if color.Bg != "" {
		style = style.Background(parseColor(color.Bg))
	}

	if color.IsSet(config.ColorAttributeBold) || color.Bold {
		style = style.Bold(color.Bold)
	}
	if color.IsSet(config.ColorAttributeItalic) || color.Italic {
		style = style.Italic(color.Italic)
	}
	if color.IsSet(config.ColorAttributeUnderline) || color.Underline {
		style = style.Underline(color.Underline)
	}
	if color.IsSet(config.ColorAttributeStrikethrough) || color.Strikethrough {
		style = style.Strikethrough(color.Strikethrough)
	}
	if color.IsSet(config.ColorAttributeReverse) || color.Reverse {
		style = style.Reverse(color.Reverse)
	}

	return style
}

func parseColor(color string) color.Color {
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
		return lipgloss.Black
	case "red":
		return lipgloss.Red
	case "green":
		return lipgloss.Green
	case "yellow":
		return lipgloss.Yellow
	case "blue":
		return lipgloss.Blue
	case "magenta":
		return lipgloss.Magenta
	case "cyan":
		return lipgloss.Cyan
	case "white":
		return lipgloss.White
	case "bright black":
		return lipgloss.BrightBlack
	case "bright red":
		return lipgloss.BrightRed
	case "bright green":
		return lipgloss.BrightGreen
	case "bright yellow":
		return lipgloss.BrightYellow
	case "bright blue":
		return lipgloss.BrightBlue
	case "bright magenta":
		return lipgloss.BrightMagenta
	case "bright cyan":
		return lipgloss.BrightCyan
	case "bright white":
		return lipgloss.BrightWhite
	default:
		if strings.HasPrefix(color, "ansi-color-") {
			code := strings.TrimPrefix(color, "ansi-color-")
			if v, err := strconv.Atoi(code); err == nil && v >= 0 && v <= 255 {
				return lipgloss.Color(code)
			}
		}
		return lipgloss.NoColor{}
	}
}
