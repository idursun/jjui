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
	"github.com/stretchr/testify/require"
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

func TestDetailsList_FilterUsesMatchesAsVisibleRows(t *testing.T) {
	list := NewDetailsList()
	list.setItems([]*item{
		{status: Modified, name: "cmd/jjui/main.go", fileName: "cmd/jjui/main.go"},
		{status: Modified, name: "internal/ui/details.go", fileName: "internal/ui/details.go", selected: true},
		{status: Modified, name: "docs/configuration.md", fileName: "docs/configuration.md"},
	})
	list.setCursor(1)

	list.setFilter("DETAILS", true)

	assert.Equal(t, 1, list.VisibleLen())
	require.NotNil(t, list.current())
	assert.Equal(t, "internal/ui/details.go", list.current().fileName)
	assert.True(t, list.files[1].selected, "filtering must preserve checked state on source items")
}

func TestDetailsList_FilterRequiresContiguousSubstringAndPreservesOrder(t *testing.T) {
	list := NewDetailsList()
	list.setItems([]*item{
		{status: Modified, name: "internal/ui/ui_test.go", fileName: "internal/ui/ui_test.go"},
		{status: Modified, name: "internal/ui/intents/details_intents.go", fileName: "internal/ui/intents/details_intents.go"},
		{status: Modified, name: "internal/ui/intents/other.go", fileName: "internal/ui/intents/other.go"},
	})
	list.setCursor(2)

	list.setFilter("intents", true)

	assert.Equal(t, 2, list.VisibleLen())
	assert.Equal(t, "internal/ui/intents/details_intents.go", list.itemAt(0).fileName)
	assert.Equal(t, "internal/ui/intents/other.go", list.itemAt(1).fileName)
	require.NotNil(t, list.current())
	assert.Equal(t, "internal/ui/intents/other.go", list.current().fileName)
}

func TestDetailsList_FilterMapsSelectionToSourceItem(t *testing.T) {
	list := NewDetailsList()
	list.setItems([]*item{
		{status: Modified, name: "one.txt", fileName: "one.txt"},
		{status: Modified, name: "two.txt", fileName: "two.txt"},
	})
	list.setFilter("two", true)

	list.rangeSelect(0, 0)

	assert.False(t, list.files[0].selected)
	assert.True(t, list.files[1].selected)
}

func TestDetailsList_FilterWithNoMatchesHasNoCurrentItem(t *testing.T) {
	list := NewDetailsList()
	list.setItems([]*item{{status: Modified, name: "file.txt", fileName: "file.txt"}})

	list.setFilter("missing", true)

	assert.Zero(t, list.VisibleLen())
	assert.Nil(t, list.current())
}
