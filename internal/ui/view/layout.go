package view

type ILayoutConstraint interface{}

type LayoutBuilder struct{}

func NewLayoutBuilder() *LayoutBuilder {
	return &LayoutBuilder{}
}

func (vm *LayoutBuilder) Container(direction ContainerDirection, containers ...ILayoutConstraint) ILayoutConstraint {
	return &LayoutConstraint{
		Type:      Container,
		Direction: direction,
		Children:  containers,
	}
}

func (vm *LayoutBuilder) VerticalContainer(containers ...ILayoutConstraint) ILayoutConstraint {
	return vm.Container(Vertical, containers...)
}

func (vm *LayoutBuilder) HorizontalContainer(containers ...ILayoutConstraint) ILayoutConstraint {
	return vm.Container(Horizontal, containers...)
}

func (vm *LayoutBuilder) Grow(view ViewId, factor int) ILayoutConstraint {
	return &LayoutConstraint{
		ViewId:     &view,
		GrowFactor: factor,
	}
}

func (vm *LayoutBuilder) Fit(view ViewId) ILayoutConstraint {
	return &LayoutConstraint{
		ViewId: &view,
		IsFit:  true,
	}
}

func (vm *LayoutBuilder) Percentage(view ViewId, percentage int) ILayoutConstraint {
	return &LayoutConstraint{
		ViewId:     &view,
		Percentage: percentage,
	}
}

func (vm *LayoutBuilder) Floating(view ViewId, anchors ...Anchor) ILayoutConstraint {
	return &LayoutConstraint{
		Type:    Floating,
		ViewId:  &view,
		Anchors: anchors,
	}
}
