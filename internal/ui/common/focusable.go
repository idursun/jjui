package common

type Editable interface {
	IsEditing() bool
}

type Overlay interface {
	IsOverlay() bool
}
