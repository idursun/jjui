package test

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// RenderImmediate renders an immediate model into a fixed-size buffer.
func RenderImmediate(model interface {
	ViewRect(dl *render.DisplayContext, box layout.Box)
}, width, height int) string {
	dl := render.NewDisplayContext()
	box := layout.NewBox(layout.Rect(0, 0, width, height))
	model.ViewRect(dl, box)
	buf := uv.NewScreenBuffer(width, height)
	dl.Render(buf)
	return buf.Render()
}
