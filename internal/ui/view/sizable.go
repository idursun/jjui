package view

type ISizeable interface {
	SetWidth(w int)
	SetHeight(h int)
}

type Sizeable struct {
	Width  int
	Height int
}

func (s *Sizeable) SetWidth(w int) {
	s.Width = w
}

func (s *Sizeable) SetHeight(h int) {
	s.Height = h
}

func NewSizeable(width int, height int) *Sizeable {
	return &Sizeable{
		Width:  width,
		Height: height,
	}
}
