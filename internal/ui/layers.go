package ui

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/annotation"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/diff"
	"github.com/idursun/jjui/internal/ui/oplog"
)

// layer is a view opened over the revisions view. Layers are ordered bottom
// to top, and the same order drives rendering, key routing and closing.
//
// A screen (oplog, diff, annotation) takes the whole view and hides the layers
// below it. A dialog is drawn over the layers below it and is the only layer
// that receives keys while it is on top.
type layer struct {
	model  common.StackedModel
	screen bool
}

func (m *Model) topLayer() (layer, bool) {
	if len(m.layers) == 0 {
		return layer{}, false
	}
	return m.layers[len(m.layers)-1], true
}

// dialog returns the top layer when it is a dialog.
func (m *Model) dialog() common.StackedModel {
	if top, ok := m.topLayer(); ok && !top.screen {
		return top.model
	}
	return nil
}

// screenIndex returns the index of the topmost screen, or -1 when the
// revisions view is the visible screen.
func (m *Model) screenIndex() int {
	for i, l := range slices.Backward(m.layers) {
		if l.screen {
			return i
		}
	}
	return -1
}

// screen returns the topmost screen, or nil when it is the revisions view.
func (m *Model) screen() common.StackedModel {
	if i := m.screenIndex(); i >= 0 {
		return m.layers[i].model
	}
	return nil
}

// primary returns the view shown as the primary pane of the split: the oplog
// when it is open, otherwise revisions.
func (m *Model) primary() common.StackedModel {
	if oplogModel, ok := findLayer[*oplog.Model](m); ok {
		return oplogModel
	}
	return m.revisions
}

// findLayer returns the topmost layer of type T.
func findLayer[T common.StackedModel](m *Model) (T, bool) {
	for _, l := range slices.Backward(m.layers) {
		if model, ok := l.model.(T); ok {
			return model, true
		}
	}
	var zero T
	return zero, false
}

// openDialog shows a dialog on top, replacing the dialog already on top.
func (m *Model) openDialog(model common.StackedModel) tea.Cmd {
	m.closeDialog()
	m.layers = append(m.layers, layer{model: model})
	return model.Init()
}

func (m *Model) closeDialog() {
	if m.dialog() != nil {
		m.layers = m.layers[:len(m.layers)-1]
	}
}

// openScreen moves a screen to the top. An open screen of the same kind is
// removed first, so there is at most one screen of each kind.
func (m *Model) openScreen(model common.StackedModel) {
	m.layers = slices.DeleteFunc(m.layers, func(l layer) bool {
		return sameScreenKind(l.model, model)
	})
	m.layers = append(m.layers, layer{model: model, screen: true})
}

func sameScreenKind(a, b common.StackedModel) bool {
	switch a.(type) {
	case *oplog.Model:
		_, ok := b.(*oplog.Model)
		return ok
	case *diff.Model:
		_, ok := b.(*diff.Model)
		return ok
	case *annotation.Model:
		_, ok := b.(*annotation.Model)
		return ok
	}
	return false
}

// closeTopLayer closes the top layer. An annotation with unsaved comments can
// refuse to close unless the close was applied.
func (m *Model) closeTopLayer(msg common.CloseViewMsg) tea.Cmd {
	top, _ := m.topLayer()
	if annotationModel, ok := top.model.(*annotation.Model); ok && !msg.Applied {
		if cmd, canClose := annotationModel.RequestClose(); !canClose {
			return cmd
		}
	}
	var cmd tea.Cmd
	if !top.screen {
		cmd = top.model.Update(msg)
	}
	m.layers = m.layers[:len(m.layers)-1]
	return cmd
}

// acceptsTargets reports whether the visible screen opens a target picker as
// a root dialog. Revisions operations open their own target pickers.
func (m *Model) acceptsTargets() bool {
	switch m.screen().(type) {
	case *diff.Model, *annotation.Model:
		return true
	}
	return false
}
