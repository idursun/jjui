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
	prefixes := strings.FieldsSeq(key)
	for prefix := range prefixes {
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
	clear(p.cache)
	p.root = nil
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
	var deferredSelectedStyles [][]string
	deferSelectedStyles := hasSelectedSubstyle(fields)
	// for a selector like "a b c", we want to inherit styles from the most specific to the least specific
	// first pass: "a b c", "a b", "a"
	// second pass: "b c", "b"
	// third pass: "c"
	start := 0
	for start < length {
		for end := length; end > start; end-- {
			selectorFields := fields[start:end]
			if deferSelectedStyles && selectorFields[len(selectorFields)-1] == "selected" {
				deferredSelectedStyles = append(deferredSelectedStyles, selectorFields)
				continue
			}
			finalStyle = finalStyle.Inherit(p.get(selectorFields...))
			if semanticFields := withoutSelectedModifier(selectorFields); len(semanticFields) > 0 {
				finalStyle = finalStyle.Inherit(p.get(semanticFields...))
			}
		}
		start++
	}
	for _, selectorFields := range deferredSelectedStyles {
		finalStyle = finalStyle.Inherit(p.get(selectorFields...))
	}
	p.cache[selector] = finalStyle
	return finalStyle
}

func hasSelectedSubstyle(fields []string) bool {
	for i, field := range fields {
		if field == "selected" && i < len(fields)-1 {
			return true
		}
	}
	return false
}

func withoutSelectedModifier(fields []string) []string {
	for i, field := range fields {
		if field != "selected" || i == len(fields)-1 {
			continue
		}
		result := make([]string, 0, len(fields)-1)
		result = append(result, fields[:i]...)
		result = append(result, fields[i+1:]...)
		return result
	}
	return nil
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
	if c == "" || c == "default" {
		return lipgloss.NoColor{}
	}
	// if it's a hex color, return it directly
	if len(c) == 7 && c[0] == '#' {
		return lipgloss.Color(c)
	}
	if index, ok := config.ParseANSIColorIndex(c); ok {
		return lipgloss.Color(strconv.Itoa(index))
	}
	return lipgloss.NoColor{}
}
