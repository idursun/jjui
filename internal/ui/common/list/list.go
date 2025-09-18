package list

import (
	"github.com/idursun/jjui/internal/models"
)

type IListProvider interface {
	CurrentItem() models.IItem
	CheckedItems() []models.IItem
}

type List[T any] struct {
	Items  []T
	Cursor int
}

func NewList[T any]() *List[T] {
	return &List[T]{
		Items:  make([]T, 0),
		Cursor: -1,
	}
}

func (l *List[T]) SetCursor(idx int) {
	if idx < 0 || idx >= len(l.Items) {
		return
	}
	l.Cursor = idx
}

func (l *List[T]) Current() T {
	var zero T
	if l.Cursor < 0 || l.Cursor >= len(l.Items) {
		return zero
	}
	return l.Items[l.Cursor]
}

func (l *List[T]) CursorDown() {
	if l.Cursor < len(l.Items)-1 {
		l.Cursor++
	}
}

func (l *List[T]) CursorUp() {
	if l.Cursor > 0 {
		l.Cursor--
	}
}

func (l *List[T]) SetItems(items []T) {
	l.Items = items
}

func (l *List[T]) AppendItems(items ...T) {
	l.Items = append(l.Items, items...)
}
