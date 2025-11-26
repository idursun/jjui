package common

import "github.com/charmbracelet/x/cellbuf"

type ISizeable interface {
	SetWidth(w int)
	SetHeight(h int)
}

var _ ISizeable = (*Sizeable)(nil)

type Sizeable struct {
	Width  int
	Height int
	Frame  cellbuf.Rectangle
}

func (s *Sizeable) SetWidth(w int) {
	s.Width = w
}

func (s *Sizeable) SetHeight(h int) {
	s.Height = h
}

func (s *Sizeable) SetFrame(f cellbuf.Rectangle) {
	s.Frame = f
	s.Width = f.Dx()
	s.Height = f.Dy()
}

func NewSizeable(width, height int) *Sizeable {
	return &Sizeable{Width: width, Height: height}
}
