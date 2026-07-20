package details

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/stretchr/testify/assert"
)

func TestDetailsList_RenderFileListShowsCheckedAndUncheckedHints(t *testing.T) {
	list := NewDetailsList()
	list.files = []*item{
		{status: Modified, name: "checked.txt", fileName: "checked.txt", selected: true},
		{status: Added, name: "unchecked.txt", fileName: "unchecked.txt"},
	}
	list.cursor = 1
	list.selectedHint = "stays as is"
	list.unselectedHint = "moves to the new revision"

	dl := render.NewDisplayContext()
	list.RenderFileList(dl, layout.NewBox(layout.Rect(0, 0, 80, 2)))
	lines := strings.Split(dl.RenderToString(80, 2), "\n")

	assert.Contains(t, lines[0], "checked.txt")
	assert.Contains(t, lines[0], "stays as is")
	assert.Contains(t, lines[1], "unchecked.txt")
	assert.Contains(t, lines[1], "moves to the new revision")
}

func TestDetailsList_SelectedDeletedKeepsStatusForeground(t *testing.T) {
	originalPalette := common.DefaultPalette
	t.Cleanup(func() { common.DefaultPalette = originalPalette })
	palette := common.NewPalette()
	palette.Update(map[string]config.Color{
		"revisions details deleted":  {Fg: "#ff5555"},
		"revisions details:selected": {Bg: "#220044"},
	})
	common.DefaultPalette = palette

	style := NewDetailsList().getStatusStyle(Deleted, true)

	assert.Equal(t, lipgloss.Color("#ff5555"), style.GetForeground())
	assert.Equal(t, lipgloss.Color("#220044"), style.GetBackground())
}
