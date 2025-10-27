package common

import (
	"strconv"
	"strings"

	"github.com/idursun/jjui/internal/config"

	"github.com/charmbracelet/lipgloss"
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
