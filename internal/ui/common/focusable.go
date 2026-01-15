package common

type Focusable interface {
	IsFocused() bool
}

type Editable interface {
	IsEditing() bool
}

type Overlay interface {
	IsOverlay() bool
}
