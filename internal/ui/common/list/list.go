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
