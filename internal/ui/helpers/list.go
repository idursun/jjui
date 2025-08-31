package helpers

type ILoadable interface {
	HasMore() bool
	RequestMore()
}

type CursorChangedHandlerFunc func()

type CursorChangedHandler struct {
	handlers []CursorChangedHandlerFunc
}

func (c *CursorChangedHandler) Notify() {
	for _, handler := range c.handlers {
		handler()
	}
}

func (c *CursorChangedHandler) AddHandler(handler CursorChangedHandlerFunc) {
	c.handlers = append(c.handlers, handler)
}

type List[T any] struct {
	*CursorChangedHandler
	Items  []T
	cursor int
}

func (l *List[T]) Cursor() int {
	return l.cursor
}

func (l *List[T]) SetCursor(cursor int) {
	l.cursor = cursor
	l.CursorChangedHandler.Notify()
}

func (l *List[T]) Prev() {
	if len(l.Items) == 0 {
		l.cursor = -1
		return
	}
	if l.cursor > 0 {
		l.SetCursor(l.cursor - 1)
	}
}

func (l *List[T]) Next() {
	if len(l.Items) == 0 {
		l.cursor = -1
		return
	}
	if l.cursor < len(l.Items)-1 {
		l.SetCursor(l.cursor + 1)
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
		Items: make([]T, 0),
		CursorChangedHandler: &CursorChangedHandler{
			handlers: make([]CursorChangedHandlerFunc, 0),
		},
		cursor: -1,
	}
}
