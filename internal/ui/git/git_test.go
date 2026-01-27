package git

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func Test_Push(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte(""))
	commandRunner.Expect(jj.GitPush("--remote", ""))
	defer commandRunner.Verify()

	op := NewModel(test.NewTestContext(commandRunner), jj.NewSelectedRevisions())
	test.SimulateModel(op, op.Init())
	_ = test.RenderImmediate(op, 100, 40)
	test.SimulateModel(op, test.Press(tea.KeyEnter))
}

func Test_Fetch(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte(""))
	commandRunner.Expect(jj.GitFetch("--remote", ""))
	defer commandRunner.Verify()

	op := NewModel(test.NewTestContext(commandRunner), jj.NewSelectedRevisions())
	test.SimulateModel(op, op.Init())
	_ = test.RenderImmediate(op, 100, 40)
	test.SimulateModel(op, test.Type("/fetch"))
	test.SimulateModel(op, test.Press(tea.KeyEnter))
	test.SimulateModel(op, test.Press(tea.KeyEnter))
}

func Test_loadBookmarks(t *testing.T) {
	const changeId = "changeid"
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkList(changeId)).SetOutput([]byte(`
feat/allow-new-bookmarks;.;false;false;false;83
feat/allow-new-bookmarks;origin;true;false;false;83
main;.;false;false;false;86
main;origin;true;false;false;86
test;.;false;false;false;d0
`))
	defer commandRunner.Verify()

	bookmarks := loadBookmarks(commandRunner, changeId)
	assert.Len(t, bookmarks, 3)
}

func Test_PushChange(t *testing.T) {
	const changeId = "abc123"
	commandRunner := test.NewTestCommandRunner(t)
	// Expect bookmark list to be loaded since we have a changeId
	commandRunner.Expect(jj.BookmarkList(changeId)).SetOutput([]byte(""))
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte(""))
	commandRunner.Expect(jj.GitPush("--change", changeId, "--remote", ""))
	defer commandRunner.Verify()

	op := NewModel(test.NewTestContext(commandRunner), jj.NewSelectedRevisions(&jj.Commit{ChangeId: changeId}))
	test.SimulateModel(op, op.Init())
	_ = test.RenderImmediate(op, 100, 40)

	// Filter for the exact item and ensure selection is at index 0
	test.SimulateModel(op, test.Type("/git push --change"))
	test.SimulateModel(op, test.Press(tea.KeyDown)) // Ensure first item is selected
	test.SimulateModel(op, test.Press(tea.KeyEnter))
	test.SimulateModel(op, test.Press(tea.KeyEnter))
}

// TestGit_ZIndex_RendersAboveMainContent verifies that the git overlay renders
// at z-index >= render.ZMenuBorder. This ensures the git operations menu
// renders above the main revision list content.
func TestGit_ZIndex_RendersAboveMainContent(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin"))

	op := NewModel(test.NewTestContext(commandRunner), jj.NewSelectedRevisions())
	test.SimulateModel(op, op.Init())

	dl := render.NewDisplayContext()
	box := layout.Box{R: cellbuf.Rect(0, 0, 100, 40)}
	op.ViewRect(dl, box)

	draws := dl.DrawList()
	assert.NotEqual(t, 0, len(draws), "Expected git overlay to produce draw operations")

	for i, draw := range draws {
		msg := fmt.Sprintf("Draw operation %d has z-index %d, expected >= %d. "+
			"Git overlay must render above main content.",
			i, draw.Z, render.ZMenuBorder)
		assert.GreaterOrEqual(t, draw.Z, render.ZMenuBorder, msg)
	}
}
