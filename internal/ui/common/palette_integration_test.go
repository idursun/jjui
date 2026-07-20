package common

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTheme_SelectedDeletedKeepsDiffForeground(t *testing.T) {
	for _, isDark := range []bool{false, true} {
		name := map[bool]string{false: "light", true: "dark"}[isDark]
		t.Run(name, func(t *testing.T) {
			theme, err := config.LoadEmbeddedTheme("default", isDark)
			require.NoError(t, err)
			theme.Colors["diff removed"] = config.Color{Fg: "red"}

			palette := NewPalette()
			palette.Update(theme.Colors)
			selectedDeleted := palette.Get("revisions", "details", "deleted", true)

			assert.Equal(t, lipgloss.Color("1"), selectedDeleted.GetForeground())
			assert.Equal(t,
				palette.Get("revisions", "details", "text", true).GetBackground(),
				selectedDeleted.GetBackground(),
			)
		})
	}
}
