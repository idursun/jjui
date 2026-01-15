package layoutview

import (
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// VStack arranges slots vertically (top to bottom).
type VStack struct {
	slots []Slot
}

// V creates a vertical stack.
func V(slots ...Slot) *VStack {
	return &VStack{slots: slots}
}

// Render lays out and renders visible slots.
func (v *VStack) Render(dl *render.DisplayContext, box layout.Box) {
	visible := visibleSlots(v.slots)
	if len(visible) == 0 {
		return
	}
	specs := make([]layout.Spec, len(visible))
	for i, slot := range visible {
		specs[i] = slot.Spec
	}
	boxes := box.V(specs...)
	for i, slot := range visible {
		if i >= len(boxes) {
			return
		}
		slot.Content.Render(dl, boxes[i])
	}
}

// Visible returns true if any slot is visible.
func (v *VStack) Visible() bool {
	for _, slot := range v.slots {
		if slot.Content != nil && slot.Content.Visible() {
			return true
		}
	}
	return false
}

// HStack arranges slots horizontally (left to right).
type HStack struct {
	slots []Slot
}

// H creates a horizontal stack.
func H(slots ...Slot) *HStack {
	return &HStack{slots: slots}
}

// Render lays out and renders visible slots.
func (h *HStack) Render(dl *render.DisplayContext, box layout.Box) {
	visible := visibleSlots(h.slots)
	if len(visible) == 0 {
		return
	}
	specs := make([]layout.Spec, len(visible))
	for i, slot := range visible {
		specs[i] = slot.Spec
	}
	boxes := box.H(specs...)
	for i, slot := range visible {
		if i >= len(boxes) {
			return
		}
		slot.Content.Render(dl, boxes[i])
	}
}

// Visible returns true if any slot is visible.
func (h *HStack) Visible() bool {
	for _, slot := range h.slots {
		if slot.Content != nil && slot.Content.Visible() {
			return true
		}
	}
	return false
}

func visibleSlots(slots []Slot) []Slot {
	visible := make([]Slot, 0, len(slots))
	for _, slot := range slots {
		if slot.Content == nil || !slot.Content.Visible() {
			continue
		}
		visible = append(visible, slot)
	}
	return visible
}
