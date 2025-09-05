package view

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
)

// Constraint defines how a view should be positioned and sized
type Constraint struct {
	// Growth factor (0 means fixed size, >0 means proportional growth)
	GrowthFactor float64
	// Whether the view's content should determine its size
	FitContent bool
	// Minimum size (width or height)
	MinSize int
	// Maximum size (width or height)
	MaxSize int
	// Fixed size (takes precedence over growth factor if > 0)
	FixedSize int
}

// LayoutDirection defines the direction of the layout (horizontal or vertical)
type LayoutDirection int

const (
	Horizontal LayoutDirection = iota
	Vertical
)

// LayoutView represents a view with its associated constraints in the layout engine
type LayoutView struct {
	View       *BaseView
	Constraint Constraint
}

// LayoutContainer is a container that manages the layout of multiple views
type LayoutContainer struct {
	*BaseView
	views     []*LayoutView
	direction LayoutDirection
	// Registry stores all views by their ID for easy lookup
	registry map[ViewId]*BaseView
}

// NewLayoutContainer creates a new layout container with the specified direction
func NewLayoutContainer(direction LayoutDirection, id ViewId) *LayoutContainer {
	m := &LayoutContainer{
		BaseView: &BaseView{
			Id:       id,
			Visible:  true,
			Focused:  false,
			Sizeable: common.NewSizeable(20, 20),
		},
		views:     make([]*LayoutView, 0),
		direction: direction,
		registry:  make(map[ViewId]*BaseView),
	}
	m.BaseView.Model = m
	m.BaseView.LayoutFn = m.Layout
	m.registry[id] = m.BaseView
	return m
}

// HorizontalContainer creates a new horizontal layout container
func HorizontalContainer(id ViewId) *LayoutContainer {
	return NewLayoutContainer(Horizontal, id)
}

// VerticalContainer creates a new vertical layout container
func VerticalContainer(id ViewId) *LayoutContainer {
	return NewLayoutContainer(Vertical, id)
}

// Add adds a view to the container with the specified constraint
func (c *LayoutContainer) Add(view *BaseView, constraint Constraint) {
	layoutView := &LayoutView{
		View:       view,
		Constraint: constraint,
	}
	c.views = append(c.views, layoutView)
	c.registry[view.Id] = view
	c.Layout()
}

// AddContainer adds a nested container to this container
func (c *LayoutContainer) AddContainer(container *LayoutContainer, constraint Constraint) {
	c.Add(container.BaseView, constraint)

	// Merge the registries
	for id, view := range container.registry {
		c.registry[id] = view
	}
}

// FindViewById searches for a view by its ID in the registry
func (c *LayoutContainer) FindViewById(id ViewId) *BaseView {
	return c.registry[id]
}

// GetRegistry returns the registry of views
func (c *LayoutContainer) GetRegistry() map[ViewId]*BaseView {
	return c.registry
}

// Remove removes a view from the container by its ID
func (c *LayoutContainer) Remove(id ViewId) {
	for i, v := range c.views {
		if v.View.Id == id {
			c.views = append(c.views[:i], c.views[i+1:]...)
			break
		}
	}
	delete(c.registry, id)
	c.Layout()
}

// Layout arranges the views according to their constraints
func (c *LayoutContainer) Layout() {
	if c.direction == Vertical {
		c.layoutVertical()
	} else {
		c.layoutHorizontal()
	}

	for _, v := range c.views {
		if v.View.LayoutFn != nil {
			v.View.LayoutFn()
		}
	}
}

func (c *LayoutContainer) layoutVertical() {
	// First pass: handle fixed size and fit content views
	remainingHeight := c.Height
	totalGrowth := 0.0

	for _, v := range c.views {
		v.View.SetWidth(c.Width)
		constraint := v.Constraint

		if constraint.FixedSize > 0 {
			// Fixed size takes precedence
			size := constraint.FixedSize
			if constraint.MaxSize > 0 && size > constraint.MaxSize {
				size = constraint.MaxSize
			}
			if constraint.MinSize > 0 && size < constraint.MinSize {
				size = constraint.MinSize
			}
			v.View.SetHeight(size)
			remainingHeight -= size
		} else if constraint.FitContent {
			// Content-based sizing
			height := lipgloss.Height(v.View.View())
			if constraint.MaxSize > 0 && height > constraint.MaxSize {
				height = constraint.MaxSize
			}
			if constraint.MinSize > 0 && height < constraint.MinSize {
				height = constraint.MinSize
			}
			v.View.SetHeight(height)
			remainingHeight -= height
		} else if constraint.GrowthFactor > 0 {
			// Track total growth factor for proportional sizing
			totalGrowth += constraint.GrowthFactor
		}
	}

	// Second pass: distribute remaining space according to growth factors
	if totalGrowth > 0 && remainingHeight > 0 {
		for _, v := range c.views {
			constraint := v.Constraint
			if constraint.FixedSize <= 0 && !constraint.FitContent && constraint.GrowthFactor > 0 {
				size := int((constraint.GrowthFactor / totalGrowth) * float64(remainingHeight))
				if constraint.MaxSize > 0 && size > constraint.MaxSize {
					size = constraint.MaxSize
				}
				if constraint.MinSize > 0 && size < constraint.MinSize {
					size = constraint.MinSize
				}
				v.View.SetHeight(size)
			}
		}
	}
}

func (c *LayoutContainer) layoutHorizontal() {
	// First pass: handle fixed size and fit content views
	remainingWidth := c.Width
	totalGrowth := 0.0

	for _, v := range c.views {
		v.View.SetHeight(c.Height)
		constraint := v.Constraint

		if constraint.FixedSize > 0 {
			// Fixed size takes precedence
			size := constraint.FixedSize
			if constraint.MaxSize > 0 && size > constraint.MaxSize {
				size = constraint.MaxSize
			}
			if constraint.MinSize > 0 && size < constraint.MinSize {
				size = constraint.MinSize
			}
			v.View.SetWidth(size)
			remainingWidth -= size
		} else if constraint.FitContent {
			// Content-based sizing
			width := lipgloss.Width(v.View.View())
			if constraint.MaxSize > 0 && width > constraint.MaxSize {
				width = constraint.MaxSize
			}
			if constraint.MinSize > 0 && width < constraint.MinSize {
				width = constraint.MinSize
			}
			v.View.SetWidth(width)
			remainingWidth -= width
		} else if constraint.GrowthFactor > 0 {
			// Track total growth factor for proportional sizing
			totalGrowth += constraint.GrowthFactor
		}
	}

	// Second pass: distribute remaining space according to growth factors
	if totalGrowth > 0 && remainingWidth > 0 {
		for _, v := range c.views {
			constraint := v.Constraint
			if constraint.FixedSize <= 0 && !constraint.FitContent && constraint.GrowthFactor > 0 {
				size := int((constraint.GrowthFactor / totalGrowth) * float64(remainingWidth))
				if constraint.MaxSize > 0 && size > constraint.MaxSize {
					size = constraint.MaxSize
				}
				if constraint.MinSize > 0 && size < constraint.MinSize {
					size = constraint.MinSize
				}
				v.View.SetWidth(size)
			}
		}
	}
}

// Init initializes all the views in the container
func (c *LayoutContainer) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, v := range c.views {
		cmds = append(cmds, v.View.Init())
	}
	return tea.Batch(cmds...)
}

// Update sends the message to all views in the container
func (c *LayoutContainer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	for _, v := range c.views {
		if !v.View.Visible {
			continue
		}

		var cmd tea.Cmd
		v.View.Model, cmd = v.View.Update(msg)
		cmds = append(cmds, cmd)
	}
	return c, tea.Batch(cmds...)
}

// View renders all views in the container
func (c *LayoutContainer) View() string {
	var views []string
	for _, v := range c.views {
		if v.View.Visible {
			log.Println("Rendering view:", v.View.Id, "Size:", v.View.Width, "x", v.View.Height)
			views = append(views, v.View.View())
		}
	}

	if c.direction == Vertical {
		return lipgloss.JoinVertical(lipgloss.Left, views...)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, views...)
}
