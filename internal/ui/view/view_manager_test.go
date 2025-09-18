package view

import (
	tea "github.com/charmbracelet/bubbletea"
	"testing"
)

// Mock view model for testing
type mockViewModel struct {
	id ViewId
}

func (m *mockViewModel) Init() tea.Cmd {
	return nil
}

func (m *mockViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *mockViewModel) View() string {
	return "mock view"
}

func (m *mockViewModel) Mount(v *ViewNode) {
	m.id = v.Id
}

func (m *mockViewModel) GetId() ViewId {
	return m.id
}

func (m *mockViewModel) SetId(id ViewId) {
	m.id = id
}

func (m *mockViewModel) GetWidth() int {
	return 10
}

func (m *mockViewModel) GetHeight() int {
	return 5
}

func (m *mockViewModel) SetWidth(width int)   {}
func (m *mockViewModel) SetHeight(height int) {}

func TestGetParentContainer(t *testing.T) {
	vm := NewViewManager()

	// Create a layout with a vertical container containing two views
	view1 := vm.CreateView(&mockViewModel{})
	view2 := vm.CreateView(&mockViewModel{})

	// Create a vertical container layout
	builder := NewLayoutBuilder()
	layout := builder.VerticalContainer(
		builder.Grow(view1.Id, 1),
		builder.Grow(view2.Id, 1),
	)

	vm.SetLayout(layout)

	// Test getting parent container for view1
	parentContainer := vm.GetParentContainer(view1.Id)
	if parentContainer == nil {
		t.Fatal("Expected to find parent container for view1")
	}

	if parentContainer.Type != Container {
		t.Fatal("Expected parent container to be of type Container")
	}

	if parentContainer.Direction != Vertical {
		t.Fatal("Expected parent container to have Vertical direction")
	}

	// Test getting parent container for view2
	parentContainer2 := vm.GetParentContainer(view2.Id)
	if parentContainer2 == nil {
		t.Fatal("Expected to find parent container for view2")
	}

	// Both views should have the same parent container
	if parentContainer != parentContainer2 {
		t.Fatal("Expected both views to have the same parent container")
	}
}

func TestGetParentContainerNested(t *testing.T) {
	vm := NewViewManager()

	// Create views
	view1 := vm.CreateView(&mockViewModel{})
	view2 := vm.CreateView(&mockViewModel{})
	view3 := vm.CreateView(&mockViewModel{})

	// Create nested layout: horizontal container with vertical container inside
	builder := NewLayoutBuilder()
	layout := builder.HorizontalContainer(
		builder.Grow(view1.Id, 1),
		builder.VerticalContainer(
			builder.Grow(view2.Id, 1),
			builder.Grow(view3.Id, 1),
		),
	)

	vm.SetLayout(layout)

	// Test that all views have parent containers
	parent1 := vm.GetParentContainer(view1.Id)
	parent2 := vm.GetParentContainer(view2.Id)
	parent3 := vm.GetParentContainer(view3.Id)

	// All views should have a parent container
	if parent1 == nil {
		t.Fatal("Expected view1 to have a parent container")
	}
	if parent2 == nil {
		t.Fatal("Expected view2 to have a parent container")
	}
	if parent3 == nil {
		t.Fatal("Expected view3 to have a parent container")
	}

	// Test that the methods work consistently
	// All views should have the same parent container
	if parent1 != parent2 || parent2 != parent3 {
		t.Fatal("Expected all views to have the same parent container")
	}

	// All parent containers should have the same direction
	if parent1.Direction != parent2.Direction || parent2.Direction != parent3.Direction {
		t.Fatal("Expected all parent containers to have the same direction")
	}
}

func TestGetViewContainer(t *testing.T) {
	vm := NewViewManager()

	// Create views
	view1 := vm.CreateView(&mockViewModel{})
	view2 := vm.CreateView(&mockViewModel{})

	// Create a vertical container layout
	builder := NewLayoutBuilder()
	layout := builder.VerticalContainer(
		builder.Grow(view1.Id, 1),
		builder.Grow(view2.Id, 1),
	)

	vm.SetLayout(layout)

	// Get the container reference for view1
	container := vm.GetViewContainer(view1.Id)
	if container == nil {
		t.Fatal("Expected to get container reference for view1")
	}

	// Verify initial direction
	if container.Direction != Vertical {
		t.Fatal("Expected container to have Vertical direction initially")
	}

	// Directly modify the container direction
	container.Direction = Horizontal

	// Verify the change took effect
	if container.Direction != Horizontal {
		t.Fatal("Expected container direction to be changed to Horizontal")
	}

	// Verify that both views now see the updated direction
	container1 := vm.GetViewContainer(view1.Id)
	container2 := vm.GetViewContainer(view2.Id)

	if container1 == nil || container2 == nil {
		t.Fatal("Expected to get container references for both views")
	}

	if container1.Direction != Horizontal || container2.Direction != Horizontal {
		t.Fatal("Expected both containers to have the updated Horizontal direction")
	}

	// Test with view that has no container
	orphanVM := NewViewManager()
	orphanView := orphanVM.CreateView(&mockViewModel{})

	orphanContainer := orphanVM.GetViewContainer(orphanView.Id)
	if orphanContainer != nil {
		t.Fatal("Expected not to get container reference for orphan view")
	}
}
