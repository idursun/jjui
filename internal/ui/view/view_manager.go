package view

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/screen"
)

type ViewManager struct {
	*Sizeable
	views               map[ViewId]*BaseView
	childViews          map[ViewId]ViewId
	focusPath           []ViewId
	editingViewId       *ViewId
	modalViewId         *ViewId
	layout              ILayoutConstraint
	floatingConstraints []ILayoutConstraint
}

func NewViewManager() *ViewManager {
	return &ViewManager{
		Sizeable:            NewSizeable(0, 0),
		views:               make(map[ViewId]*BaseView),
		childViews:          make(map[ViewId]ViewId),
		focusPath:           []ViewId{},
		layout:              nil,
		floatingConstraints: []ILayoutConstraint{},
	}
}

type ViewNode struct {
	*ViewOpts
	Parent      *BaseView
	ViewManager *ViewManager
}

type IViewModel interface {
	tea.Model
	IView
	Mount(v *ViewNode)
}

func (vm *ViewManager) CreateChildView(parentViewId ViewId, model IViewModel) *BaseView {
	parent := vm.GetView(parentViewId)
	if parent == nil && parentViewId != "root" {
		panic("parent view not found")
	}
	size := NewSizeable(0, 0)
	node := &ViewNode{
		Parent:      parent,
		ViewManager: vm,
		ViewOpts:    &ViewOpts{Visible: true, Sizeable: size},
	}

	model.Mount(node)

	view := &BaseView{
		Model:    model,
		ViewOpts: node.ViewOpts,
	}
	vm.RegisterView(view)
	vm.childViews[parentViewId] = node.Id
	return view
}

func (vm *ViewManager) CreateView(model IViewModel) *BaseView {
	return vm.CreateChildView("root", model)
}

func (vm *ViewManager) RegisterView(view *BaseView) {
	vm.views[view.Id] = view
}

func (vm *ViewManager) UnregisterView(viewId ViewId) {
	delete(vm.views, viewId)
	// Remove from focus path if present
	for i, id := range vm.focusPath {
		if id == viewId {
			vm.focusPath = append(vm.focusPath[:i], vm.focusPath[i+1:]...)
			break
		}
	}
	if vm.editingViewId != nil && *vm.editingViewId == viewId {
		vm.editingViewId = nil
	}
}

func (vm *ViewManager) GetView(viewId ViewId) *BaseView {
	return vm.views[viewId]
}

func (vm *ViewManager) GetChildView(parentViewId ViewId) *BaseView {
	if id, exists := vm.childViews[parentViewId]; exists {
		return vm.views[id]
	}
	return nil
}

func (vm *ViewManager) FocusView(viewId ViewId) {
	// Remove if already in focus path
	for i, id := range vm.focusPath {
		if id == viewId {
			vm.focusPath = append(vm.focusPath[:i], vm.focusPath[i+1:]...)
			break
		}
	}
	// Add to the end of the focus path
	vm.focusPath = append(vm.focusPath, viewId)
}

func (vm *ViewManager) GetFocusedView() *BaseView {
	if len(vm.focusPath) == 0 {
		return nil
	}
	lastFocusedId := vm.focusPath[len(vm.focusPath)-1]
	return vm.views[lastFocusedId]
}

func (vm *ViewManager) GetFocusedViews() []*BaseView {
	var ret []*BaseView
	for _, id := range vm.focusPath {
		if v, exists := vm.views[id]; exists {
			ret = append(ret, v)
		}
	}
	return ret
}

func (vm *ViewManager) IsEditing() bool {
	return vm.editingViewId != nil
}

func (vm *ViewManager) StartEditing(viewId ViewId) {
	if v, exists := vm.views[viewId]; exists {
		vm.editingViewId = &viewId
		if e, ok := v.Model.(Editable); ok {
			e.OnEdit()
		}
	}
}

func (vm *ViewManager) StopEditing() {
	vm.editingViewId = nil
}

func (vm *ViewManager) GetViews() []*BaseView {
	var ret []*BaseView
	for _, v := range vm.views {
		ret = append(ret, v)
	}
	return ret
}

func (vm *ViewManager) GetEditingView() *BaseView {
	if vm.editingViewId == nil {
		return nil
	}
	return vm.views[*vm.editingViewId]
}

func (vm *ViewManager) AddModal(view *BaseView, anchors ...Anchor) {
	vm.modalViewId = &view.Id
	vm.RegisterView(view)
	vm.FocusView(view.Id)

	// Create a floating constraint for the modal
	builder := NewLayoutBuilder()
	floatingConstraint := builder.Floating(view.Id, anchors...)

	// Add the floating constraint to the layout
	if vm.layout != nil {
		// If we have an existing layout, we need to add the floating constraint
		// For now, we'll store it separately and handle it in the render method
		vm.floatingConstraints = append(vm.floatingConstraints, floatingConstraint)
	}
}

func (vm *ViewManager) SetLayout(root ILayoutConstraint) {
	vm.layout = root
}

// UpdateViewConstraint updates the constraint for a specific view in the layout tree
func (vm *ViewManager) UpdateViewConstraint(viewId ViewId, updateFunc func(*LayoutConstraint)) bool {
	if vm.layout == nil {
		return false
	}
	return vm.updateConstraintInTree(vm.layout, viewId, updateFunc)
}

// updateConstraintInTree recursively searches and updates constraints in the layout tree
func (vm *ViewManager) updateConstraintInTree(constraint ILayoutConstraint, viewId ViewId, updateFunc func(*LayoutConstraint)) bool {
	if layoutConstraint, ok := constraint.(*LayoutConstraint); ok {
		// If this constraint matches the view ID, update it
		if layoutConstraint.ViewId != nil && *layoutConstraint.ViewId == viewId {
			updateFunc(layoutConstraint)
			return true
		}

		// Recursively search in children
		for _, child := range layoutConstraint.Children {
			if vm.updateConstraintInTree(child, viewId, updateFunc) {
				return true
			}
		}
	}
	return false
}

// hasDirectChild checks if a container has the specified view as a direct child
func (vm *ViewManager) hasDirectChild(constraint *LayoutConstraint, viewId ViewId) bool {
	// Check if any direct child is the target view
	for _, child := range constraint.Children {
		if childConstraint, ok := child.(*LayoutConstraint); ok {
			// Only check direct children, not nested containers
			if childConstraint.ViewId != nil && *childConstraint.ViewId == viewId {
				return true
			}
		}
	}
	return false
}

// GetParentContainer finds and returns the parent container that contains the specified view
func (vm *ViewManager) GetParentContainer(viewId ViewId) *LayoutConstraint {
	if vm.layout == nil {
		return nil
	}
	return vm.findParentContainerInTree(vm.layout, viewId)
}

// findParentContainerInTree recursively searches for a container that contains the specified view
func (vm *ViewManager) findParentContainerInTree(constraint ILayoutConstraint, viewId ViewId) *LayoutConstraint {
	if layoutConstraint, ok := constraint.(*LayoutConstraint); ok {
		// If this is a container, check if it directly contains the target view
		if layoutConstraint.Type == Container {
			if vm.hasDirectChild(layoutConstraint, viewId) {
				return layoutConstraint
			}
		}

		// Recursively search in children
		for _, child := range layoutConstraint.Children {
			if result := vm.findParentContainerInTree(child, viewId); result != nil {
				return result
			}
		}
	}
	return nil
}

// GetViewContainer returns a reference to the container that contains the specified view
// This allows direct manipulation of the container's properties
func (vm *ViewManager) GetViewContainer(viewId ViewId) *LayoutConstraint {
	return vm.GetParentContainer(viewId)
}

func (vm *ViewManager) calculateAnchors(constraint *LayoutConstraint, view *BaseView) (int, int) {
	var left, top int
	viewWidth := view.Width
	viewHeight := view.Height

	// Default to center if no anchors
	anchors := constraint.Anchors
	if len(anchors) == 0 {
		anchors = []Anchor{CenterX(), CenterY()}
	}

	hasHorizontal := false
	hasVertical := false

	for _, anchor := range anchors {
		switch anchor.Type {
		case AnchorTypeLeft:
			left = anchor.Offset
			hasHorizontal = true
		case AnchorTypeRight:
			left = vm.Width - viewWidth - anchor.Offset
			hasHorizontal = true
		case AnchorTypeTop:
			top = anchor.Offset
			hasVertical = true
		case AnchorTypeBottom:
			top = vm.Height - viewHeight - anchor.Offset
			hasVertical = true
		case AnchorTypeCenterX:
			left = (vm.Width - viewWidth) / 2
			hasHorizontal = true
		case AnchorTypeCenterY:
			top = (vm.Height - viewHeight) / 2
			hasVertical = true
		}
	}

	if !hasHorizontal {
		left = (vm.Width - viewWidth) / 2
	}
	if !hasVertical {
		top = (vm.Height - viewHeight) / 2
	}

	return left, top
}

func (vm *ViewManager) Render() string {
	if vm.layout == nil {
		return ""
	}
	vm.Layout()

	rendered := vm.renderLayout(vm.layout)

	// Render floating constraints (including modals)
	for _, floatingConstraint := range vm.floatingConstraints {
		if constraint, ok := floatingConstraint.(*LayoutConstraint); ok && constraint.Type == Floating {
			if constraint.ViewId != nil {
				view := vm.GetView(*constraint.ViewId)
				if view != nil && view.Visible {
					viewContent := view.Model.View()
					view.SetWidth(lipgloss.Width(viewContent))
					view.SetHeight(lipgloss.Height(viewContent))
					left, top := vm.calculateAnchors(constraint, view)
					rendered = screen.Stacked(rendered, viewContent, left, top)
				}
			}
		}
	}

	return rendered
}

func (vm *ViewManager) renderLayout(layout ILayoutConstraint) string {
	if constraint, ok := layout.(*LayoutConstraint); ok {
		// If this is a view reference
		if constraint.ViewId != nil {
			view := vm.views[*constraint.ViewId]
			if view != nil && view.Visible {
				return view.Model.View()
			}
			return ""
		}

		// If this is a container
		var contents []string
		for _, child := range constraint.Children {
			childContent := vm.renderLayout(child)
			contents = append(contents, childContent)
		}
		if constraint.Type == Container {
			if constraint.Direction == Vertical {
				return lipgloss.JoinVertical(0, contents...)
			} else {
				return lipgloss.JoinHorizontal(lipgloss.Left, contents...)
			}
		}
	}
	return ""
}

func (vm *ViewManager) RestorePreviousFocus() {
	if len(vm.focusPath) > 1 {
		// Remove the current focused view
		vm.focusPath = vm.focusPath[:len(vm.focusPath)-1]
		if vm.modalViewId != nil {
			// Remove the floating constraint for this modal
			vm.removeFloatingConstraint(*vm.modalViewId)
			vm.UnregisterView(*vm.modalViewId)
			vm.modalViewId = nil
		}
	}
}

func (vm *ViewManager) IsFocused(id ViewId) bool {
	if len(vm.focusPath) == 0 {
		return false
	}
	return vm.focusPath[len(vm.focusPath)-1] == id
}

func (vm *ViewManager) IsThisEditing(id ViewId) bool {
	if vm.editingViewId == nil {
		return false
	}
	return *vm.editingViewId == id
}

func (vm *ViewManager) Layout() {
	if vm.layout == nil {
		return
	}

	// Calculate and apply layout constraints
	vm.calculateLayout(vm.layout, vm.Width, vm.Height)
}

// calculateLayout calculates the sizes for all views based on the constraints
func (vm *ViewManager) calculateLayout(layout ILayoutConstraint, availWidth, availHeight int) (int, int) {
	constraint, ok := layout.(*LayoutConstraint)
	if !ok {
		return 0, 0
	}

	// If this is a view reference, set its size
	if constraint.ViewId != nil {
		view := vm.views[*constraint.ViewId]
		if view != nil && view.Visible {
			// Set the view size based on the available space and constraint
			if constraint.IsFit {
				view.SetWidth(availWidth)
				_, preferredHeight := lipgloss.Size(view.Model.View())
				if view.Height > 0 {
					preferredHeight = view.Height
				}

				// Don't exceed available height
				view.SetHeight(min(availHeight, preferredHeight))
			} else if constraint.Percentage > 0 {
				// For "Percentage" constraint, use the specified percentage of available space
				// The percentage should be applied in the container's direction
				// For now, we'll use the full available space and let the container handle the percentage
				view.SetWidth(availWidth)
				view.SetHeight(availHeight)
			} else {
				// For "Grow" constraint, use the available space and growth factor
				view.SetWidth(availWidth)
				view.SetHeight(availHeight)
			}
			return view.Width, view.Height
		}
		return 0, 0
	}

	// If this is a container
	if len(constraint.Children) == 0 {
		return 0, 0
	}

	// First pass: identify fit views, percentage views, and calculate total growth factor
	var totalGrowth int
	var fitViews, percentageViews, growViews []*LayoutConstraint
	var visibleViews []*LayoutConstraint

	for _, child := range constraint.Children {
		childConstraint, ok := child.(*LayoutConstraint)
		if !ok {
			continue
		}

		// Check if the view is visible
		if childConstraint.ViewId != nil {
			view := vm.views[*childConstraint.ViewId]
			if view == nil || !view.Visible {
				continue
			}
		}

		visibleViews = append(visibleViews, childConstraint)

		if childConstraint.IsFit {
			fitViews = append(fitViews, childConstraint)
		} else if childConstraint.Percentage > 0 {
			percentageViews = append(percentageViews, childConstraint)
		} else {
			growFactor := max(childConstraint.GrowFactor, 1) // Default to 1 if not specified
			totalGrowth += growFactor
			growViews = append(growViews, childConstraint)
		}
	}

	// Handle based on container orientation
	if constraint.Type == Container {
		if constraint.Direction == Vertical {
			return vm.layoutVertical(fitViews, percentageViews, growViews, totalGrowth, availWidth, availHeight)
		} else {
			return vm.layoutHorizontal(fitViews, percentageViews, growViews, totalGrowth, availWidth, availHeight)
		}
	}

	// If not a container, return zero dimensions
	return 0, 0
}

// layoutVertical handles vertical container layout
func (vm *ViewManager) layoutVertical(fitViews, percentageViews, growViews []*LayoutConstraint, totalGrowth, availWidth, availHeight int) (int, int) {
	// First, calculate sizes for fit views (fixed height)
	var usedHeight int
	for _, fitView := range fitViews {
		_, childHeight := vm.calculateLayout(fitView, availWidth, availHeight-usedHeight)
		usedHeight += childHeight
	}

	// Calculate sizes for percentage views
	for _, percentageView := range percentageViews {
		percentageHeight := (availHeight * percentageView.Percentage) / 100
		vm.calculateLayout(percentageView, availWidth, percentageHeight)
		usedHeight += percentageHeight
	}

	// Distribute remaining space to grow views
	remainingHeight := availHeight - usedHeight
	if remainingHeight <= 0 || len(growViews) == 0 {
		return availWidth, usedHeight
	}

	var actualUsedHeight int
	for i, growView := range growViews {
		growFactor := max(growView.GrowFactor, 1)
		var childHeight int

		if i == len(growViews)-1 {
			// Last view gets all remaining space to avoid rounding issues
			childHeight = remainingHeight - actualUsedHeight
		} else {
			// Calculate proportional height based on growth factor
			childHeight = (remainingHeight * growFactor) / totalGrowth
		}

		if childHeight > 0 {
			vm.calculateLayout(growView, availWidth, childHeight)
			actualUsedHeight += childHeight
		}
	}

	return availWidth, usedHeight + actualUsedHeight
}

// layoutHorizontal handles horizontal container layout
func (vm *ViewManager) layoutHorizontal(fitViews, percentageViews, growViews []*LayoutConstraint, totalGrowth, availWidth, availHeight int) (int, int) {
	// First, calculate sizes for fit views (fixed width)
	var usedWidth int
	for _, fitView := range fitViews {
		childWidth, _ := vm.calculateLayout(fitView, availWidth-usedWidth, availHeight)
		usedWidth += childWidth
	}

	// Calculate sizes for percentage views
	for _, percentageView := range percentageViews {
		percentageWidth := (availWidth * percentageView.Percentage) / 100
		vm.calculateLayout(percentageView, percentageWidth, availHeight)
		usedWidth += percentageWidth
	}

	// Distribute remaining space to grow views
	remainingWidth := availWidth - usedWidth
	if remainingWidth <= 0 || len(growViews) == 0 {
		return usedWidth, availHeight
	}

	var actualUsedWidth int
	for i, growView := range growViews {
		growFactor := max(growView.GrowFactor, 1)
		var childWidth int

		if i == len(growViews)-1 {
			// Last view gets all remaining space to avoid rounding issues
			childWidth = remainingWidth - actualUsedWidth
		} else {
			// Calculate proportional width based on growth factor
			childWidth = (remainingWidth * growFactor) / totalGrowth
		}

		if childWidth > 0 {
			vm.calculateLayout(growView, childWidth, availHeight)
			actualUsedWidth += childWidth
		}
	}

	return usedWidth + actualUsedWidth, availHeight
}

func (vm *ViewManager) FocusedCount() int {
	return len(vm.focusPath)
}

func (vm *ViewManager) GetViewsNeedRefresh() []*BaseView {
	var ret []*BaseView
	for _, view := range vm.views {
		if view.NeedsRefresh {
			ret = append(ret, view)
		}
	}
	return ret
}

func (vm *ViewManager) removeFloatingConstraint(viewId ViewId) {
	for i, constraint := range vm.floatingConstraints {
		if layoutConstraint, ok := constraint.(*LayoutConstraint); ok {
			if layoutConstraint.ViewId != nil && *layoutConstraint.ViewId == viewId {
				vm.floatingConstraints = append(vm.floatingConstraints[:i], vm.floatingConstraints[i+1:]...)
				break
			}
		}
	}
}
