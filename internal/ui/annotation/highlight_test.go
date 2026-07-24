package annotation

import (
	"strconv"
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

func TestChangedWordHighlightUsesCompleteLineContext(t *testing.T) {
	highlighter := newSourceHighlighter(true)
	wordBackground := lipgloss.NewStyle().Background(lipgloss.Color("2"))
	rendered := highlighter.renderChanged("example.go", `// changed`, `// original`, wordBackground)

	buffer := uv.NewScreenBuffer(10, 1)
	uv.NewStyledString(rendered).Draw(buffer, layout.Rect(0, 0, 10, 1))
	commentCell := buffer.CellAt(1, 0)
	changedCell := buffer.CellAt(3, 0)
	require.NotNil(t, commentCell)
	require.NotNil(t, changedCell)
	require.NotNil(t, commentCell.Style.Fg)
	require.NotNil(t, changedCell.Style.Fg)
	assert.Equal(t, commentCell.Style.Fg, changedCell.Style.Fg)
}

func TestHighlightCacheRetainsTheCurrentSource(t *testing.T) {
	highlighter := newSourceHighlighter(true)
	for index := 0; index < 320; index++ {
		highlighter.highlight("example.go", "value "+strconv.Itoa(index))
	}

	assert.Len(t, highlighter.cache, 320)
	assert.Equal(t, 320, highlighter.highlightMisses)

	highlighter.resetSource()
	assert.Empty(t, highlighter.cache)
}
