package common

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
)

type Focusable interface {
	IsFocused() bool
}

type Editable interface {
	IsEditing() bool
}

type Overlay interface {
	IsOverlay() bool
}

type IMouseAware interface {
	Update(msg tea.Msg) tea.Cmd
	ClickAt(x, y int) tea.Cmd
	Scroll(delta int) tea.Cmd
}

type MouseAware struct {
	dragging  bool
	dragStart cellbuf.Position
}

func (m *MouseAware) ClickAt(x, y int) tea.Cmd {
	return nil
}

func (m *MouseAware) Scroll(delta int) tea.Cmd {
	return nil
}

func NewMouseAware() *MouseAware {
	return &MouseAware{}
}
