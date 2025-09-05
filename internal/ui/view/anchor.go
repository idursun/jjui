package view

type AnchorType int

const (
	AnchorTypeLeft AnchorType = iota
	AnchorTypeRight
	AnchorTypeTop
	AnchorTypeBottom
	AnchorTypeCenterX
	AnchorTypeCenterY
)

type Anchor struct {
	Type   AnchorType
	Offset int
}

func Left(offset ...int) Anchor {
	o := 0
	if len(offset) > 0 {
		o = offset[0]
	}
	return Anchor{Type: AnchorTypeLeft, Offset: o}
}

func Right(offset ...int) Anchor {
	o := 0
	if len(offset) > 0 {
		o = offset[0]
	}
	return Anchor{Type: AnchorTypeRight, Offset: o}
}

func Top(offset ...int) Anchor {
	o := 0
	if len(offset) > 0 {
		o = offset[0]
	}
	return Anchor{Type: AnchorTypeTop, Offset: o}
}

func Bottom(offset ...int) Anchor {
	o := 0
	if len(offset) > 0 {
		o = offset[0]
	}
	return Anchor{Type: AnchorTypeBottom, Offset: o}
}

func CenterX() Anchor {
	return Anchor{Type: AnchorTypeCenterX}
}

func CenterY() Anchor {
	return Anchor{Type: AnchorTypeCenterY}
}
