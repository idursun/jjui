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
	test.SimulateModel(op, func() tea.Msg { return intents.GitOpenFilter{} })
	test.SimulateModel(op, test.Type("git push --change"))
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func Test_PushSelectedBookmarks(t *testing.T) {
	const changeId1 = "abc123"
	const changeId2 = "def456"
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkList(changeId1)).SetOutput([]byte("feature-a;.;false;false;false;83\n"))
	commandRunner.Expect(jj.BookmarkList(changeId2)).SetOutput([]byte("feature-b;.;false;false;false;86\n"))
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte(""))
	commandRunner.Expect(jj.GitPush("--remote", "", "--bookmark", "feature-a", "--bookmark", "feature-b"))
	defer commandRunner.Verify()

	selected := jj.NewSelectedRevisions(&jj.Commit{ChangeId: changeId1}, &jj.Commit{ChangeId: changeId2})
	op := NewModel(test.NewTestContext(commandRunner), selected)
	test.SimulateModel(op, op.Init())
	_ = test.RenderImmediate(op, 100, 40)

	test.SimulateModel(op, intents.Invoke(intents.GitFilter{Kind: intents.GitFilterPush}))
	test.SimulateModel(op, intents.Invoke(intents.GitApplyShortcut{Key: "b"}))
}

func Test_PushSelectedBookmarks_SkipsRemoteOnlyAndNonMatchingRemotes(t *testing.T) {
	const (
		localOnOrigin   = "abc123"
		remoteOnly      = "def456"
		localOnUpstream = "ghi789"
		newLocal        = "jkl012"
	)
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkList(localOnOrigin)).SetOutput([]byte("feature-a;.;true;false;false;83\nfeature-a;origin;true;false;false;83\n"))
	commandRunner.Expect(jj.BookmarkList(remoteOnly)).SetOutput([]byte("feature-b;origin;false;false;false;86\n"))
	commandRunner.Expect(jj.BookmarkList(localOnUpstream)).SetOutput([]byte("feature-c;.;true;false;false;90\nfeature-c;upstream;true;false;false;90\n"))
	commandRunner.Expect(jj.BookmarkList(newLocal)).SetOutput([]byte("feature-d;.;false;false;false;92\n"))
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin\nupstream\n"))
	commandRunner.Expect(jj.GitPush("--remote", "origin", "--bookmark", "feature-a", "--bookmark", "feature-d"))
	defer commandRunner.Verify()

	selected := jj.NewSelectedRevisions(
		&jj.Commit{ChangeId: localOnOrigin},
		&jj.Commit{ChangeId: remoteOnly},
		&jj.Commit{ChangeId: localOnUpstream},
		&jj.Commit{ChangeId: newLocal},
	)
	op := NewModel(test.NewTestContext(commandRunner), selected)
	test.SimulateModel(op, op.Init())
	_ = test.RenderImmediate(op, 100, 40)

	test.SimulateModel(op, intents.Invoke(intents.GitFilter{Kind: intents.GitFilterPush}))
	test.SimulateModel(op, intents.Invoke(intents.GitApplyShortcut{Key: "b"}))
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
