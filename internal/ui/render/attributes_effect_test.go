package render

import (
	"testing"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/stretchr/testify/assert"
)

func TestAttributesEffectUnderlinesOnlyTheRectAndKeepsColors(t *testing.T) {
	buffer := uv.NewScreenBuffer(4, 1)
	uv.NewStyledString(lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Render("abcd")).Draw(buffer, layout.Rect(0, 0, 4, 1))
	fg := buffer.CellAt(1, 0).Style.Fg

	AttributesEffect{Rect: layout.Rect(1, 0, 2, 1), Style: lipgloss.NewStyle().Underline(true)}.Apply(buffer)

	assert.Equal(t, uv.UnderlineStyle(0), buffer.CellAt(0, 0).Style.Underline)
	for x := 1; x <= 2; x++ {
		cell := buffer.CellAt(x, 0)
		assert.Equal(t, uv.UnderlineSingle, cell.Style.Underline)
		assert.Equal(t, fg, cell.Style.Fg)
		assert.NotZero(t, cell.Style.Attrs&uv.AttrBold, "existing attributes are kept")
	}
	assert.Equal(t, uv.UnderlineStyle(0), buffer.CellAt(3, 0).Style.Underline)
}
