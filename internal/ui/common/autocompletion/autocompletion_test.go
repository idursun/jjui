package autocompletion

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestWithStyleScope(t *testing.T) {
	palette := theme.NewPalette()
	palette.Update(map[string]config.Color{
		":selected":       {Fg: "red"},
		"matched":         {Fg: "white"},
		"revset:selected": {Fg: "green"},
		"revset matched":  {Fg: "blue"},
	})

	originalPalette := theme.DefaultPalette
	theme.DefaultPalette = palette
	defer func() { theme.DefaultPalette = originalPalette }()

	scoped := &AutoCompletionInput{}
	WithStyleScope("revset")(scoped)
	assert.Equal(t, lipgloss.Color("2"), scoped.Styles.Selected.GetForeground())
	assert.Equal(t, lipgloss.Color("4"), scoped.Styles.Matched.GetForeground())

	generic := &AutoCompletionInput{}
	WithStyleScope("")(generic)
	assert.Equal(t, lipgloss.Color("1"), generic.Styles.Selected.GetForeground())
	assert.Equal(t, lipgloss.Color("7"), generic.Styles.Matched.GetForeground())
}
