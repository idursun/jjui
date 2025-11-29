package common

import "github.com/charmbracelet/x/cellbuf"

type IViewNode interface {
	GetViewNode() *ViewNode
}

var _ IViewNode = (*ViewNode)(nil)

type ViewNode struct {
	Parent *ViewNode
	Width  int
	Height int
	Frame  cellbuf.Rectangle
}

func (s *ViewNode) GetViewNode() *ViewNode {
	return s
}

func (s *ViewNode) SetWidth(w int) {
	s.Width = w
}

func (s *ViewNode) SetHeight(h int) {
	s.Height = h
}

func (s *ViewNode) SetFrame(f cellbuf.Rectangle) {
	s.Frame = f
	s.SetWidth(f.Dx())
	s.SetHeight(f.Dy())
}

func (s *ViewNode) ToLocal(x, y int) (int, int) {
	return x - s.Frame.Min.X, y - s.Frame.Min.Y
}

func NewViewNode(width, height int) *ViewNode {
	return &ViewNode{Width: width, Height: height}
}
