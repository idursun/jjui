package details

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetailsList_ToggleTreeView_PreservesSelection(t *testing.T) {
	list := NewDetailsList(styles{})
	items := []*item{
		{fileName: "src/a.go", name: "src/a.go", status: Modified},
		{fileName: "src/b.go", name: "src/b.go", status: Added},
		{fileName: "lib/c.go", name: "lib/c.go", status: Deleted},
	}

	list.setItems(items)
	list.setCursor(1)
	selectedBefore := list.current().fileName

	list.ToggleTreeView()
	assert.Equal(t, selectedBefore, list.current().fileName)

	list.ToggleTreeView()
	assert.Equal(t, selectedBefore, list.current().fileName)
}

func TestDetailsList_TreeVisibleCount(t *testing.T) {
	list := NewDetailsList(styles{})
	items := []*item{
		{fileName: "src/a.go", name: "src/a.go", status: Modified},
		{fileName: "src/b.go", name: "src/b.go", status: Added},
		{fileName: "lib/c.go", name: "lib/c.go", status: Deleted},
	}

	list.setItems(items)
	assert.Equal(t, 3, list.Len())

	list.ToggleTreeView()
	assert.Equal(t, 5, list.Len())
}
