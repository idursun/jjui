package models

type IItem interface {
	Equals(other IItem) bool
}

type ICheckable interface {
	IsChecked() bool
	Toggle()
	SetChecked(checked bool)
}

type Checkable struct {
	Checked bool
}

func (c *Checkable) IsChecked() bool {
	return c.Checked
}

func (c *Checkable) SetChecked(checked bool) {
	c.Checked = checked
}

func (c *Checkable) Toggle() {
	c.Checked = !c.Checked
}
