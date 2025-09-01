package models

type ICheckable interface {
	IsChecked() bool
	Toggle()
}

type Checkable struct {
	checked bool
}

func (c *Checkable) IsChecked() bool {
	return c.checked
}

func (c *Checkable) Toggle() {
	c.checked = !c.checked
}
