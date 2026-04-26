package git

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/intents"
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
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func Test_Fetch(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte(""))
	commandRunner.Expect(jj.GitFetch("--remote", ""))
	defer commandRunner.Verify()

	op := NewModel(test.NewTestContext(commandRunner), jj.NewSelectedRevisions())
	test.SimulateModel(op, op.Init())
	_ = test.RenderImmediate(op, 100, 40)
	test.SimulateModel(op, func() tea.Msg { return intents.GitOpenFilter{} })
	test.SimulateModel(op, test.Type("fetch"))
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func Test_FilterIntentPressedTwice_ExecutesShortcut(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte(""))
	commandRunner.Expect(jj.GitFetch("--remote", ""))
	defer commandRunner.Verify()

	op := NewModel(test.NewTestContext(commandRunner), jj.NewSelectedRevisions())
	test.SimulateModel(op, op.Init())
	_ = test.RenderImmediate(op, 100, 40)

	// First press applies the category filter; second press executes its shortcut.
	test.SimulateModel(op, func() tea.Msg { return intents.GitFilter{Kind: intents.GitFilterFetch} })
	test.SimulateModel(op, func() tea.Msg { return intents.GitFilter{Kind: intents.GitFilterFetch} })
}

func Test_loadBookmarks(t *testing.T) {
	const changeId = "changeid"
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkList(changeId)).SetOutput([]byte(`
feat/allow-new-bookmarks;.;true;false;false;false;83
feat/allow-new-bookmarks;origin;true;true;false;false;83
main;.;true;false;false;false;86
main;origin;true;true;false;false;86
test;.;true;false;false;false;d0
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
	test.SimulateModel(op, func() tea.Msg { return intents.GitOpenFilter{} })
	test.SimulateModel(op, test.Type("git push --change"))
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func Test_NewModel_DoesNotPanicWithNilSelectedRevision(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte(""))
	defer commandRunner.Verify()

	assert.NotPanics(t, func() {
		model := NewModel(test.NewTestContext(commandRunner), jj.NewSelectedRevisions(nil))
		assert.NotNil(t, model)
	})
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
	box := layout.Box{R: layout.Rect(0, 0, 100, 40)}
	dl.AddDraw(box.R, strings.Repeat("x", box.R.Dx()*box.R.Dy()), render.ZBase)
	op.ViewRect(dl, box)

	rendered := dl.RenderToString(box.R.Dx(), box.R.Dy())
	assert.Contains(t, rendered, "Remotes:", "git overlay should remain visible above base content")
}
