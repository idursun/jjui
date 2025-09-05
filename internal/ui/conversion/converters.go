// This package is disabled for now to avoid import cycles. Uncomment when needed.
package conversion

/*
import (
	"github.com/idursun/jjui/internal/ui"
	"github.com/idursun/jjui/internal/ui/layout"
)

// ViewSpecToConstraint converts an old ViewSpec to a new Constraint
func ViewSpecToConstraint(spec *ui.ViewSpec) layout.Constraint {
	if spec.Grow {
		return layout.Grow(1.0)
	}
	if spec.Content {
		return layout.FitContent()
	}
	return layout.Constraint{}
}

// ContainerToLayoutContainer converts an old Container to a new LayoutContainer
func ContainerToLayoutContainer(container *ui.Container) *layout.LayoutContainer {
	var layoutContainer *layout.LayoutContainer
	if container.IsVertical() {
		layoutContainer = layout.VerticalContainer(container.BaseView.Id)
	} else {
		layoutContainer = layout.HorizontalContainer(container.BaseView.Id)
	}

	// Set size
	layoutContainer.SetWidth(container.Width)
	layoutContainer.SetHeight(container.Height)

	// Add views
	for _, viewSpec := range container.GetViews() {
		layoutContainer.Add(viewSpec.View, ViewSpecToConstraint(viewSpec))
	}

	return layoutContainer
}

// StackedContainerToStackedLayoutContainer converts an old StackedContainer to a new StackedLayoutContainer
func StackedContainerToStackedLayoutContainer(container *ui.StackedContainer) *layout.StackedLayoutContainer {
	layoutContainer := layout.NewStackedLayoutContainer(container.BaseView.Id)

	// Set size
	layoutContainer.SetWidth(container.Width)
	layoutContainer.SetHeight(container.Height)

	// Add views
	for _, viewSpec := range container.GetViews() {
		layoutContainer.Add(viewSpec.View, ViewSpecToConstraint(viewSpec))
	}

	return layoutContainer
}
*/
