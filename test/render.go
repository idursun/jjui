package test

import (
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// RenderImmediate renders an immediate model into a fixed-size buffer.
func RenderImmediate(model interface {
	ViewRect(dl *render.DisplayContext, box layout.Box)
}, width, height int) string {
	dl := render.NewDisplayContext()
	box := layout.NewBox(cellbuf.Rect(0, 0, width, height))
	model.ViewRect(dl, box)
	screen := cellbuf.NewBuffer(width, height)
	dl.Render(screen)
	return cellbuf.Render(screen)
}
