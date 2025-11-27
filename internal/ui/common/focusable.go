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

type Draggable interface {
	DragStart(x, y int) bool
	DragMove(x, y int) tea.Cmd
	DragEnd(x, y int) tea.Cmd
	IsDragging() bool
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

type DragAware struct {
	dragging  bool
	dragStart cellbuf.Position
}

func (d *DragAware) DragStart(_ int, _ int) bool {
	return false
}

func (d *DragAware) DragMove(_ int, _ int) tea.Cmd {
	return nil
}

func (d *DragAware) DragEnd(_ int, _ int) tea.Cmd {
	d.dragging = false
	return nil
}

func (d *DragAware) IsDragging() bool {
	return d.dragging
}

func (d *DragAware) BeginDrag(x, y int) {
	d.dragging = true
	d.dragStart = cellbuf.Pos(x, y)
}

func (d *DragAware) DragStartPosition() cellbuf.Position {
	return d.dragStart
}

func NewDragAware() *DragAware {
	return &DragAware{}
}
