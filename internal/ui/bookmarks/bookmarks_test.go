package bookmarks

import (
	"slices"
	"testing"

	"github.com/idursun/jjui/internal/ui/common/menu"
	"github.com/stretchr/testify/assert"
)

func TestDistanceMap(t *testing.T) {
	selectedCommitId := "x"
	changeIds := []string{"a", "x", "b", "c", "d"}
	distanceMap := calcDistanceMap(selectedCommitId, changeIds)
	assert.Equal(t, 0, distanceMap["x"])
	assert.Equal(t, -1, distanceMap["a"])
	assert.Equal(t, 1, distanceMap["b"])
	assert.Equal(t, 2, distanceMap["c"])
	assert.Equal(t, 3, distanceMap["d"])
	assert.Equal(t, 0, distanceMap["nonexistent"])
}

func Test_Sorting_MoveCommands(t *testing.T) {
	items := []menu.Item{
		item{name: "move feature", dist: 5, priority: moveCommand},
		item{name: "move main", dist: 1, priority: moveCommand},
		item{name: "move very-old-feature", dist: 15, priority: moveCommand},
		item{name: "move backwards", dist: -2, priority: moveCommand},
	}
	slices.SortFunc(items, itemSorter)
	var sorted []string
	for _, i := range items {
		sorted = append(sorted, i.(item).name)
	}
	assert.Equal(t, []string{"move main", "move feature", "move very-old-feature", "move backwards"}, sorted)
}

func Test_Sorting_MixedCommands(t *testing.T) {
	items := []menu.Item{
		item{name: "move very-old-feature", dist: 2, priority: moveCommand},
		item{name: "move main", dist: 0, priority: moveCommand},
		item{name: "delete very-old-feature", dist: 3, priority: deleteCommand},
		item{name: "delete main", dist: 0, priority: deleteCommand},
	}
	slices.SortFunc(items, itemSorter)
	var sorted []string
	for _, i := range items {
		sorted = append(sorted, i.(item).name)
	}
	assert.Equal(t, []string{"move main", "move very-old-feature", "delete main", "delete very-old-feature"}, sorted)
}
