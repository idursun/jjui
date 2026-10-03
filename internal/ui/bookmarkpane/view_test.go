package bookmarkpane

import (
	"image/color"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderUsesBookmarkThemeAndConsistentPadding(t *testing.T) {
	previous := theme.DefaultPalette
	t.Cleanup(func() { theme.DefaultPalette = previous })
	resolved, err := config.LoadEmbeddedTheme("default", true)
	require.NoError(t, err)
	theme.DefaultPalette = theme.NewPalette()
	theme.DefaultPalette.Update(resolved.Colors)

	model := New(nil)
	for _, tc := range []struct {
		name  string
		state filterState
	}{
		{"off", filterOff},
		{"editing", filterEditing},
		{"applied", filterApplied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := tc.state
			model.filterState = state
			model.filterText = "feature"
			model.filterInput.SetValue("feature")
			dl := render.NewDisplayContext()
			model.ViewRect(dl, layout.NewBox(layout.Rect(2, 3, 40, 10)))
			screen := render.NewScreenBuffer(44, 16)
			dl.Render(screen)

			title := screen.CellAt(3, 3)
			assert.Equal(t, "B", title.Content)
			assert.Equal(t, ansi.IndexedColor(230), title.Style.Fg)
			assert.Equal(t, ansi.IndexedColor(62), title.Style.Bg)
			assert.NotZero(t, title.Style.Attrs&uv.AttrBold)
			assert.Equal(t, " ", screen.CellAt(3, 4).Content, "blank line below title")

			remoteTitle := screen.CellAt(3, 5)
			assert.Equal(t, "R", remoteTitle.Content)
			assert.Equal(t, ansi.Magenta, remoteTitle.Style.Fg)
			assert.Nil(t, remoteTitle.Style.Bg, "remote prompt must not inherit title background")
			assert.Equal(t, ansi.Cyan, screen.CellAt(12, 5).Style.Fg, "active remote")
			assert.NotZero(t, screen.CellAt(12, 5).Style.Attrs&uv.AttrBold)

			if state != filterOff {
				assert.Equal(t, "F", screen.CellAt(3, 6).Content, "editing and applied filter align with headers")
			}
		})
	}
}

func TestViewRefreshesStylesAfterPaletteChange(t *testing.T) {
	previous := theme.DefaultPalette
	t.Cleanup(func() { theme.DefaultPalette = previous })
	theme.DefaultPalette = theme.NewPalette()
	theme.DefaultPalette.Update(map[string]config.Color{
		"bookmarks": {Fg: "#ff0000"},
	})
	model := New(nil)
	model.Update(rowsLoadedMsg{tree: loadBookmarkTree(
		"feature;.;true;false;false;false;abc123\n", nil, "", nil,
	)})
	theme.DefaultPalette.Update(map[string]config.Color{
		"bookmarks":       {Fg: "#00ff00"},
		"bookmarks title": {Bg: "#0000ff"},
	})

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 10)))
	screen := render.NewScreenBuffer(40, 10)
	dl.Render(screen)
	assert.Equal(t, color.RGBA{B: 255, A: 255}, screen.CellAt(1, 0).Style.Bg)
	name := screen.CellAt(12, 4)
	assert.Equal(t, "f", name.Content)
	assert.Equal(t, color.RGBA{G: 255, A: 255}, name.Style.Fg)
}

func TestRowSelectionStylesTextAndBackgroundOnlyWhenFocused(t *testing.T) {
	previous := theme.DefaultPalette
	t.Cleanup(func() { theme.DefaultPalette = previous })
	theme.DefaultPalette = theme.NewPalette()
	theme.DefaultPalette.Update(map[string]config.Color{
		"picker text":               {Fg: "#ff0000"},
		"revisions:selected":        {Bg: "#ff0000"},
		"bookmarks":                 {Fg: "#eeeeee"},
		"bookmarks:selected":        {Fg: "#00ffff", Bg: "#808080", Bold: new(true)},
		"bookmarks matched":         {Fg: "#ff00ff"},
		"bookmarks dimmed:selected": {Fg: "#ffff00"},
	})
	// Blending #808080 halfway toward black yields #5b5b5b, not the raw gray.
	theme.DefaultPalette.ConfigureBackgroundBlend(0.5, "#000000", nil)

	model := New(nil)
	model.Update(rowsLoadedMsg{tree: loadBookmarkTree(
		"feature;.;true;false;false;false;abc123\nfeature;origin;true;true;false;false;def456\n",
		map[string]bool{"feature": true}, "", nil,
	)})
	require.Len(t, model.visibleRows, 2)

	for _, focused := range []bool{false, true} {
		model.SetFocused(focused)
		for index, nameColumn := range []int{12, 16} {
			model.cursor = index
			dl := render.NewDisplayContext()
			model.renderListRow(dl, index, layout.Rect(0, 0, 80, 1))
			screen := render.NewScreenBuffer(80, 1)
			dl.Render(screen)
			name := screen.CellAt(nameColumn, 0)
			require.Equal(t, "f", name.Content)
			if focused {
				assert.Equal(t, color.RGBA{R: 0, G: 255, B: 255, A: 255}, name.Style.Fg)
				assert.NotZero(t, name.Style.Attrs&uv.AttrBold)
				assert.Equal(t, color.RGBA{R: 91, G: 91, B: 91, A: 255}, screen.CellAt(79, 0).Style.Bg, "selection background spans row and is blended")
			} else {
				assert.Equal(t, color.RGBA{R: 238, G: 238, B: 238, A: 255}, name.Style.Fg)
				assert.Zero(t, name.Style.Attrs&uv.AttrBold)
				assert.Nil(t, screen.CellAt(79, 0).Style.Bg)
			}
			if index == 1 {
				assert.Equal(t, color.RGBA{R: 255, G: 0, B: 255, A: 255}, screen.CellAt(7, 0).Style.Fg, "remote keeps semantic matched color")
				if focused {
					assert.Equal(t, "d", screen.CellAt(39, 0).Content)
					assert.Equal(t, color.RGBA{R: 255, G: 255, B: 0, A: 255}, screen.CellAt(39, 0).Style.Fg, "selected metadata respects its role override")
				}
			}
		}
	}
}
