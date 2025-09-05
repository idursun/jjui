package view

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
)

// StackedLayoutContainer is a container that only renders the top-most view
type StackedLayoutContainer struct {
	*BaseView
	views    []*LayoutView
	registry map[ViewId]*BaseView
}

// NewStackedLayoutContainer creates a new stacked layout container
func NewStackedLayoutContainer(id ViewId) *StackedLayoutContainer {
	m := &StackedLayoutContainer{
		BaseView: &BaseView{
			Id:       id,
			Visible:  true,
			Focused:  false,
			Sizeable: common.NewSizeable(20, 20),
		},
		views:    make([]*LayoutView, 0),
		registry: make(map[ViewId]*BaseView),
	}
	m.BaseView.Model = m
	m.BaseView.LayoutFn = m.Layout
	m.registry[id] = m.BaseView
	return m
}

// Add adds a view to the container with the specified constraint
func (c *StackedLayoutContainer) Add(view *BaseView, constraint Constraint) {
	layoutView := &LayoutView{
		View:       view,
		Constraint: constraint,
	}
	c.views = append(c.views, layoutView)
	c.registry[view.Id] = view
	c.Layout()
}

// Push adds a view to the top of the stack
func (c *StackedLayoutContainer) Push(view *BaseView, constraint Constraint) {
	c.Add(view, constraint)
}

// Pop removes and returns the top-most view from the stack
func (c *StackedLayoutContainer) Pop() *BaseView {
	if len(c.views) == 0 {
		return nil
	}
	lastIndex := len(c.views) - 1
	popped := c.views[lastIndex].View
	c.views = c.views[:lastIndex]
	delete(c.registry, popped.Id)
	c.Layout()
	return popped
}

// Top returns the top-most view without removing it from the stack
func (c *StackedLayoutContainer) Top() *BaseView {
	if len(c.views) == 0 {
		return nil
	}
	return c.views[len(c.views)-1].View
}

// FindViewById searches for a view by its ID in the registry
func (c *StackedLayoutContainer) FindViewById(id ViewId) *BaseView {
	return c.registry[id]
}

// GetRegistry returns the registry of views
func (c *StackedLayoutContainer) GetRegistry() map[ViewId]*BaseView {
	return c.registry
}

// Layout sets the size of the top-most view and calls its layout function
func (c *StackedLayoutContainer) Layout() {
	top := c.Top()
	if top != nil {
		top.SetWidth(c.Width)
		top.SetHeight(c.Height)
		if top.LayoutFn != nil {
			top.LayoutFn()
		}
	}
}

// Init initializes the top-most view
func (c *StackedLayoutContainer) Init() tea.Cmd {
	top := c.Top()
	if top != nil {
		return top.Init()
	}
	return nil
}

// Update sends the message to the top-most view
func (c *StackedLayoutContainer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	top := c.Top()
	if top != nil {
		var cmd tea.Cmd
		top.Model, cmd = top.Update(msg)
		return c, cmd
	}
	return c, nil
}

// View renders the top-most view
func (c *StackedLayoutContainer) View() string {
	top := c.Top()
	if top != nil && top.Visible {
		return top.View()
	}
	return ""
}
