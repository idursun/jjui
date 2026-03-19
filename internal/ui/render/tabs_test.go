package render

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestExpandTabs_UsesTabStops(t *testing.T) {
	t.Cleanup(func() {
		SetWidthMethod(ansi.WcWidth)
	})

	assert.Equal(t, "a   b", ExpandTabs("a\tb"))
	assert.Equal(t, "ab  b", ExpandTabs("ab\tb"))
	assert.Equal(t, "abc b", ExpandTabs("abc\tb"))
	assert.Equal(t, "abcd    b", ExpandTabs("abcd\tb"))
	assert.Equal(t, "+   foo", ExpandTabs("+\tfoo"))
	assert.Equal(t, "++  foo", ExpandTabs("++\tfoo"))
}

func TestExpandTabs_PreservesAnsi(t *testing.T) {
	assert.Equal(t, "\x1b[31m+   foo\x1b[0m", ExpandTabs("\x1b[31m+\tfoo\x1b[0m"))
}
