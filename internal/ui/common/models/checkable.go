package models

type ICheckable interface {
	IsChecked() bool
	Toggle()
}

type Checkable struct {
	Checked bool
}

func (c *Checkable) IsChecked() bool {
	return c.Checked
}

func (c *Checkable) Toggle() {
	c.Checked = !c.Checked
}
