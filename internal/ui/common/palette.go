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

	if color.Bold != nil {
		style = style.Bold(*color.Bold)
	}
	if color.Italic != nil {
		style = style.Italic(*color.Italic)
	}
	if color.Underline != nil {
		style = style.Underline(*color.Underline)
	}
	if color.Strikethrough != nil {
		style = style.Strikethrough(*color.Strikethrough)
	}
	if color.Reverse != nil {
		style = style.Reverse(*color.Reverse)
	}

	return style
}

func parseColor(c string) color.Color {
	// if it's a hex color, return it directly
	if len(c) == 7 && c[0] == '#' {
		return lipgloss.Color(c)
	}
	// if it's an ANSI256 color, return it directly
	if v, err := strconv.Atoi(c); err == nil {
		if v >= 0 && v <= 255 {
			return lipgloss.Color(c)
		}
	}
	// otherwise, try to parse it as a named color
	switch c {
	case "black":
		return lipgloss.Color("0")
	case "red":
		return lipgloss.Color("1")
	case "green":
		return lipgloss.Color("2")
	case "yellow":
		return lipgloss.Color("3")
	case "blue":
		return lipgloss.Color("4")
	case "magenta":
		return lipgloss.Color("5")
	case "cyan":
		return lipgloss.Color("6")
	case "white":
		return lipgloss.Color("7")
	case "bright black":
		return lipgloss.Color("8")
	case "bright red":
		return lipgloss.Color("9")
	case "bright green":
		return lipgloss.Color("10")
	case "bright yellow":
		return lipgloss.Color("11")
	case "bright blue":
		return lipgloss.Color("12")
	case "bright magenta":
		return lipgloss.Color("13")
	case "bright cyan":
		return lipgloss.Color("14")
	case "bright white":
		return lipgloss.Color("15")
	default:
		if strings.HasPrefix(c, "ansi-color-") {
			code := strings.TrimPrefix(c, "ansi-color-")
			if v, err := strconv.Atoi(code); err == nil && v >= 0 && v <= 255 {
				return lipgloss.Color(code)
			}
		}
		return lipgloss.NoColor{}
	}
}
