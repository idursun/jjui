package layoutview

import (
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// Slot represents a position in the layout that can hold content.
type Slot struct {
	Spec    layout.Spec
	Content SlotContent
}

// SlotContent is anything that can be rendered in a slot.
type SlotContent interface {
	Render(dl *render.DisplayList, box layout.Box)
	Visible() bool
}

// Fixed creates a slot with fixed size.
func Fixed(size int, content SlotContent) Slot {
	return Slot{Spec: layout.Fixed(size), Content: content}
}

// Fill creates a slot that fills remaining space with given weight.
func Fill(weight int, content SlotContent) Slot {
	return Slot{Spec: layout.Fill(float64(weight)), Content: content}
}

// Percent creates a slot taking percentage of available space.
func Percent(pct int, content SlotContent) Slot {
	return Slot{Spec: layout.Percent(pct), Content: content}
}

type emptySlot struct{}

func (e emptySlot) Render(_ *render.DisplayList, _ layout.Box) {}
func (e emptySlot) Visible() bool {
	return true
}

// Empty creates an empty slot (for spacing or placeholders).
func Empty() SlotContent {
	return emptySlot{}
}
