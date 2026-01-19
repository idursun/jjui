package bookmarks

import (
	"fmt"
	"slices"
	"testing"

	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common/menu"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
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

// TestBookmarks_ZIndex_RendersAboveMainContent verifies that the bookmarks
// overlay renders at z-index >= menu.ZIndexBorder. This ensures the bookmarks
// operations menu renders above the main revision list content.
func TestBookmarks_ZIndex_RendersAboveMainContent(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin"))
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(""))
	commandRunner.Expect(jj.BookmarkListMovable("abc123")).SetOutput([]byte(""))

	commit := &jj.Commit{ChangeId: "abc123", CommitId: "commit123"}
	op := NewModel(test.NewTestContext(commandRunner), commit, []string{"commit123"})
	test.SimulateModel(op, op.Init())

	dl := render.NewDisplayContext()
	box := layout.Box{R: cellbuf.Rect(0, 0, 100, 40)}
	op.ViewRect(dl, box)

	draws := dl.DrawList()

	for i, draw := range draws {
		msg := fmt.Sprintf("Draw operation %d has z-index %d, expected >= %d. "+
			"Bookmarks overlay must render above main content.",
			i, draw.Z, menu.ZIndexBorder)
		assert.GreaterOrEqual(t, draw.Z, menu.ZIndexBorder, msg)
	}
}
