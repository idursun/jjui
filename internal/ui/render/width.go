package render

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

var widthMethod = ansi.WcWidth

// SetWidthMethod updates the render width method.
func SetWidthMethod(m ansi.Method) {
	widthMethod = m
}

// WidthMethod returns the current render width method.
func WidthMethod() ansi.Method {
	return widthMethod
}

// StringWidth returns the display width of a string.
func StringWidth(s string) int {
	return widthMethod.StringWidth(s)
}

// BlockWidth returns the display width of the widest line in a multiline string.
func BlockWidth(s string) int {
	width := 0
	for line := range strings.SplitSeq(s, "\n") {
		width = max(width, StringWidth(line))
	}
	return width
}

// NewScreenBuffer creates a screen buffer using the current width method.
func NewScreenBuffer(w, h int) uv.ScreenBuffer {
	buf := uv.NewScreenBuffer(w, h)
	buf.Method = widthMethod
	return buf
}
