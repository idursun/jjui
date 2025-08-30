package helpers

type ILoadable interface {
	HasMore() bool
	RequestMore()
}

type List[T any] struct {
	Items  []T
	Cursor int
}

func (l *List[T]) Prev() {
	if len(l.Items) == 0 {
		l.Cursor = -1
		return
	}
	if l.Cursor > 0 {
		l.Cursor--
	}
}

func (l *List[T]) Next() {
	if len(l.Items) == 0 {
		l.Cursor = -1
		return
	}
	if l.Cursor < len(l.Items)-1 {
		l.Cursor++
	}
}

func (l *List[T]) Current() *T {
	if l.Cursor >= 0 && l.Cursor < len(l.Items) {
		return &l.Items[l.Cursor]
	}
	return nil
}

func NewList[T any]() *List[T] {
	return &List[T]{
		Items:  make([]T, 0),
		Cursor: -1,
	}
}
