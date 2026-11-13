package annotation

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/idursun/jjui/internal/config"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/theme"
	"github.com/stretchr/testify/assert"
)

func TestAnnotationContrastAfterDiffAndSelectionComposition(t *testing.T) {
	previous := theme.DefaultPalette
	t.Cleanup(func() { theme.DefaultPalette = previous })
	for _, tt := range []struct {
		name, surface, text, syntax, added, deleted, selected string
		dark                                                  bool
	}{
		{"screenshot", "#282a35", "#f8f8f3", "#b695f3", "#88ef8d", "#ef6d67", "#383e56", true},
		{"light", "#ffffff", "#24292f", "#8250df", "#228822", "#cc3333", "#c0c0c0", false},
		{"dark saturated", "#000000", "#ffffff", "#000080", "#00ff00", "#ff0000", "#808080", true},
		{"light pale syntax", "#fffaf0", "#222222", "#aaddaa", "#66dd66", "#ff8888", "#dddddd", false},
		{"midgrey", "#777777", "#ffffff", "#888888", "#88aa88", "#aa8888", "#999999", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			theme.DefaultPalette = theme.NewPalette()
			theme.DefaultPalette.Update(map[string]config.Color{
				"syntax text": {Fg: tt.text}, "syntax keyword": {Fg: tt.syntax},
				"syntax string": {Fg: tt.syntax}, "syntax function": {Fg: tt.syntax},
				"syntax comment": {Fg: tt.syntax}, "syntax type": {Fg: tt.syntax},
				"added": {Fg: tt.added}, "deleted": {Fg: tt.deleted},
				"annotation:selected": {Bg: tt.selected},
			})
			theme.DefaultPalette.ConfigureBackgroundBlend(0.4, tt.surface, nil)
			model := mouseTestModel(4)
			model.context = &appContext.MainContext{TerminalBackground: tt.surface, TerminalHasDarkBackground: tt.dark}
			lines := model.document.files[0].Patch.Lines
			lines[0] = patchLine{Kind: lineRemoved, OldLine: 1, Content: `return "old"`, PairContent: `return "new"`}
			lines[1] = patchLine{Kind: lineAdded, NewLine: 1, Content: `return "new"`, PairContent: `return "old"`}
			lines[2].Content = `return "unchanged"`
			lines[3].Content = `return "selected"`
			model.cursor = 3
			buffer := uv.NewScreenBuffer(50, 6)
			display := render.NewDisplayContext()
			model.ViewRect(display, layout.NewBox(layout.Rect(0, 0, 50, 6)))
			display.Render(buffer)
			rgb := func(c ansi.Color) theme.RGB { v, _ := theme.ResolveRGBFallback(c, nil); return v }
			for _, y := range []int{1, 2, 4} {
				checked := 0
				for x := contentColumn; x < 40; x++ {
					cell := buffer.CellAt(x, y)
					if cell.Content == " " || cell.Style.Fg == nil || cell.Style.Bg == nil {
						continue
					}
					checked++
					assert.GreaterOrEqual(t, theme.Ratio(rgb(cell.Style.Fg), rgb(cell.Style.Bg)), theme.Minimum, "x=%d y=%d", x, y)
				}
				assert.Positive(t, checked)
			}
			assert.Nil(t, buffer.CellAt(contentColumn, 3).Style.Bg, "unhighlighted context should retain its styling")
			assert.NotEqual(t, buffer.CellAt(contentColumn, 1).Style.Bg, buffer.CellAt(contentColumn+8, 1).Style.Bg, "word and line backgrounds remain distinct")
			assert.NotEqual(t, buffer.CellAt(contentColumn, 2).Style.Bg, buffer.CellAt(contentColumn+8, 2).Style.Bg, "word and line backgrounds remain distinct")
		})
	}
}
