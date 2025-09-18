package view

type LayoutType int

const (
	Container LayoutType = iota
	Floating
)

type ContainerDirection int

const (
	Vertical ContainerDirection = iota
	Horizontal
)

type LayoutConstraint struct {
	Type       LayoutType
	Direction  ContainerDirection // For Container type
	Children   []ILayoutConstraint
	ViewId     *ViewId
	GrowFactor int
	IsFit      bool
	Percentage int // For Percentage constraint (0-100)
	Anchors    []Anchor
}

func (lc *LayoutConstraint) GetViewId() *ViewId {
	return lc.ViewId
}
