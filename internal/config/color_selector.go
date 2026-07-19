package config

import "strings"

type ColorVariant string

const SelectedVariant ColorVariant = "selected"

type ColorSelector struct {
	fields            []string
	variant           ColorVariant
	usesVariantSuffix bool
}

// ParseColorSelector accepts both "text:selected" and the legacy "selected text" syntax.
func ParseColorSelector(value string) ColorSelector {
	fields := strings.Fields(value)
	selector := ColorSelector{fields: fields}
	if len(fields) == 0 {
		return selector
	}

	last := fields[len(fields)-1]
	if last == ":"+string(SelectedVariant) {
		selector.fields = fields[:len(fields)-1]
		selector.variant = SelectedVariant
		selector.usesVariantSuffix = true
		return selector
	}
	if base, ok := strings.CutSuffix(last, ":"+string(SelectedVariant)); ok && base != "" {
		selector.fields = append([]string(nil), fields...)
		selector.fields[len(selector.fields)-1] = base
		selector.variant = SelectedVariant
		selector.usesVariantSuffix = true
		return selector
	}

	for i, field := range fields {
		if field != string(SelectedVariant) {
			continue
		}
		selector.fields = make([]string, 0, len(fields)-1)
		selector.fields = append(selector.fields, fields[:i]...)
		selector.fields = append(selector.fields, fields[i+1:]...)
		selector.variant = SelectedVariant
		return selector
	}

	return selector
}

func (s ColorSelector) Fields() []string {
	return append([]string(nil), s.fields...)
}

func (s ColorSelector) HasVariant(variant ColorVariant) bool {
	return s.variant == variant
}

func (s ColorSelector) Key() string {
	if s.variant == "" {
		return strings.Join(s.fields, " ")
	}
	if len(s.fields) == 0 {
		return ":" + string(s.variant)
	}
	fields := append([]string(nil), s.fields...)
	fields[len(fields)-1] += ":" + string(s.variant)
	return strings.Join(fields, " ")
}

// NormalizeColorSelectors returns a copy keyed by normalized selectors.
// When both spellings occur in one layer, the suffix spelling wins.
func NormalizeColorSelectors(colors map[string]Color) map[string]Color {
	if colors == nil {
		return nil
	}
	result := make(map[string]Color, len(colors))
	for key, color := range colors {
		selector := ParseColorSelector(key)
		if !selector.usesVariantSuffix {
			result[selector.Key()] = color
		}
	}
	for key, color := range colors {
		selector := ParseColorSelector(key)
		if selector.usesVariantSuffix {
			result[selector.Key()] = color
		}
	}
	return result
}
