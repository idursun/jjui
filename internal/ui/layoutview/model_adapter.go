package layoutview

import (
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type visibleModel interface {
	Visible() bool
}

// ModelSlot wraps an ImmediateModel to implement SlotContent.
type ModelSlot struct {
	Model   common.ImmediateModel
	visible func() bool
}

// Model wraps an ImmediateModel for use in slots.
func Model(m common.ImmediateModel) *ModelSlot {
	return &ModelSlot{Model: m}
}

// ModelWhen wraps an ImmediateModel with a visibility condition.
func ModelWhen(m common.ImmediateModel, visible func() bool) *ModelSlot {
	return &ModelSlot{Model: m, visible: visible}
}

func (m *ModelSlot) Render(dl *render.DisplayList, box layout.Box) {
	if m == nil || m.Model == nil {
		return
	}
	m.Model.ViewRect(dl, box)
}

func (m *ModelSlot) Visible() bool {
	if m == nil || m.Model == nil {
		return false
	}
	if m.visible != nil {
		return m.visible()
	}
	if vm, ok := m.Model.(visibleModel); ok {
		return vm.Visible()
	}
	return true
}
