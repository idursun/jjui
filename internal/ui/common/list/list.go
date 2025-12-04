package list

import "io"

type IList interface {
	Len() int
	GetItemRenderer(index int) IItemRenderer
}

type IListCursor interface {
	Cursor() int
	SetCursor(index int)
}

// CursorWriter provides write access plus positional information. Implementors
// can query where they are writing in both local item coordinates and the
// current viewport.
type CursorWriter interface {
	io.Writer
	LocalPos() (line, col int)
	ViewportPos() (line, col int)
}

type IItemRenderer interface {
	Render(w CursorWriter, width int)
	Height() int
}
