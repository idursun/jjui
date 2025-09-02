package list

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

func (l *List[T]) Current() T {
	var zero T
	if l.Cursor < 0 || l.Cursor >= len(l.Items) {
		return zero
	}
	return l.Items[l.Cursor]
}

func (l *List[T]) SetItems(items []T) {
	l.Items = items
}
