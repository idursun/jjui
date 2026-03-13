package render

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestBlockWidthUsesWidestLine(t *testing.T) {
	t.Cleanup(func() {
		SetWidthMethod(ansi.WcWidth)
	})

	for _, method := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
		SetWidthMethod(method)
		assert.Equal(t, 3, BlockWidth("ab\n中d"))
		assert.Equal(t, 5, StringWidth("ab\n中d"))
	}
}
