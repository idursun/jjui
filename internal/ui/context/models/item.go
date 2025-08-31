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

type Checkable interface {
	Toggle()
	IsChecked() bool
}
