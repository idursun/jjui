package annotation

import (
	"testing"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangedWordBackgroundPreservesSyntaxForeground(t *testing.T) {
	highlighter := newSourceHighlighter(true)
	wordBackground := lipgloss.NewStyle().Background(lipgloss.Color("2"))
	rendered := highlighter.renderChanged("example.go", `return "new"`, `return "old"`, wordBackground)

	buffer := uv.NewScreenBuffer(12, 1)
	uv.NewStyledString(rendered).Draw(buffer, layout.Rect(0, 0, 12, 1))
	changedCell := buffer.CellAt(8, 0)
	require.NotNil(t, changedCell)
	require.NotNil(t, changedCell.Style.Fg)
	require.NotNil(t, changedCell.Style.Bg)
	changedWordBackground := changedCell.Style.Bg

	render.HighlightEffect{
		Rect:  layout.Rect(0, 0, 12, 1),
		Style: lipgloss.NewStyle().Background(lipgloss.Color("52")),
	}.Apply(buffer)

	plainCell := buffer.CellAt(0, 0)
	changedCell = buffer.CellAt(8, 0)
	assert.NotNil(t, plainCell.Style.Bg)
	assert.NotNil(t, changedCell.Style.Fg)
	assert.Equal(t, changedWordBackground, changedCell.Style.Bg)
}
