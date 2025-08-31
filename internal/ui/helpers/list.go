package helpers

type ILoadable interface {
	HasMore() bool
	RequestMore()
}

type List[T any] struct {
	Items  []T
	cursor int
}

func (l *List[T]) Cursor() int {
	return l.cursor
}

func (l *List[T]) SetCursor(cursor int) {
	l.cursor = cursor
}

func (l *List[T]) Prev() {
	if len(l.Items) == 0 {
		l.cursor = -1
		return
	}
	if l.cursor > 0 {
		l.cursor--
	}
}

func (l *List[T]) Next() {
	if len(l.Items) == 0 {
		l.cursor = -1
		return
	}
	if l.cursor < len(l.Items)-1 {
		l.cursor++
	}
}

func (l *List[T]) Current() T {
	var zero T
	if l.cursor >= 0 && l.cursor < len(l.Items) {
		return l.Items[l.cursor]
	}
	return zero
}

func NewList[T any]() *List[T] {
	return &List[T]{
		Items:  make([]T, 0),
		cursor: -1,
	}
}
