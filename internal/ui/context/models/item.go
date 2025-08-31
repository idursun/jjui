package models

type ItemKind uint

var (
	Revision ItemKind
	File     ItemKind
	OpLog    ItemKind
	Evolog   ItemKind
)

type BaseItem struct {
	Kind ItemKind
}

type ICheckable interface {
	Toggle()
	IsChecked() bool
}

type Checkable struct {
	ICheckable
	checked bool
}

func (c *Checkable) IsChecked() bool {
	return c.checked
}

func (c *Checkable) Toggle() {
	c.checked = !c.checked
}
