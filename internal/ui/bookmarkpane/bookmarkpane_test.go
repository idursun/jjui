package bookmarkpane

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bookmarkListOutput(count int) string {
	var out strings.Builder
	for i := range count {
		fmt.Fprintf(&out, "bookmark-%02d;.;true;false;false;false;commit-%02d\n", i, i)
	}
	return out.String()
}

func TestOpen_SortsLocalBookmarksFirstByDistanceAndSelectsClosestMoveable(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(
		"remote-only;origin;true;true;false;false;ccc333\ncurrent;.;true;false;false;false;bbb222\nfar-local;.;true;false;false;false;ddd444\nnear-local;.;true;false;false;false;aaa111\n",
	))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("bbb222")
	model.SetVisibleCommitIDs([]string{"aaa111", "bbb222", "ccc333", "ddd444"})
	test.SimulateModel(model, model.Open())

	require.True(t, model.Visible())
	require.Len(t, model.visibleRows, 4)

	assert.Equal(t, "current", model.visibleRows[0].Node.Target())
	assert.Equal(t, "far-local", model.visibleRows[1].Node.Target())
	assert.Equal(t, "near-local", model.visibleRows[2].Node.Target())
	assert.Equal(t, "remote-only@origin", model.visibleRows[3].Node.Target())

	target, ok := model.selectedTarget()
	require.True(t, ok)
	assert.Equal(t, "current", target)
	assert.Equal(t, "bbb222", model.selectedCommitID())
}

func TestRenameSelected_LocalBookmarkOpensPrompt(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	cmd := model.Update(intents.BookmarkViewRename{})
	require.NotNil(t, cmd)
	showInput, ok := cmd().(common.ShowInputMsg)
	require.True(t, ok, "rename should request input")
	assert.Equal(t, "main", showInput.InitialValue)
}

func TestCreateSelected_ReturnsBeginCreateMessage(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	cmd := model.Update(intents.BookmarkViewCreate{})
	require.NotNil(t, cmd)
	msg, ok := cmd().(BeginCreateBookmarkMsg)
	require.True(t, ok)
	assert.Equal(t, BeginCreateBookmarkMsg{}, msg)
}

func TestCreateSelected_ReturnsBeginCreateMessageWhenNoBookmarksVisible(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(""))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	cmd := model.Update(intents.BookmarkViewCreate{})
	require.NotNil(t, cmd)
	msg, ok := cmd().(BeginCreateBookmarkMsg)
	require.True(t, ok)
	assert.Equal(t, BeginCreateBookmarkMsg{}, msg)
}

func TestRevealSelected_ReturnsRevealMessageWithCommitID(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	model.SetVisibleCommitIDs([]string{"abc123"})
	test.SimulateModel(model, model.Open())

	cmd := model.Update(intents.BookmarkViewReveal{})
	require.NotNil(t, cmd)
	msg, ok := cmd().(RevealBookmarkMsg)
	require.True(t, ok)
	assert.Equal(t, "abc123", msg.CommitID)
}

func TestRevealSelected_WhenAlreadyAtBookmark_ReturnsRevealMessage(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("abc123")
	model.SetVisibleCommitIDs([]string{"abc123"})
	test.SimulateModel(model, model.Open())

	cmd := model.Update(intents.BookmarkViewReveal{})
	require.NotNil(t, cmd)
	msg, ok := cmd().(RevealBookmarkMsg)
	require.True(t, ok)
	assert.Equal(t, "abc123", msg.CommitID)
}

func TestBookmarkViewPageNavigation_UsesRenderedListHeight(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(bookmarkListOutput(20)))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.Open())

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 10)))
	require.Equal(t, 6, model.lastListHeight, "list height should reflect rendered viewport height")
	require.Equal(t, 0, model.listRenderer.GetScrollOffset())
	require.Equal(t, 0, model.listRenderer.GetFirstRowIndex())

	model.Update(intents.BookmarkViewNavigate{Delta: 1, IsPage: true})
	assert.Equal(t, 6, model.listRenderer.GetScrollOffset())

	dl = render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 10)))
	assert.Equal(t, 6, model.listRenderer.GetFirstRowIndex())
	assert.Equal(t, 11, model.listRenderer.GetLastRowIndex())
}

func TestOpen_ResetsCachedPageHeightBeforeFirstRender(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(bookmarkListOutput(20)))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.Open())
	model.ViewRect(render.NewDisplayContext(), layout.NewBox(layout.Rect(0, 0, 40, 10)))
	require.Equal(t, 6, model.lastListHeight)

	model.Close()
	require.Zero(t, model.lastListHeight)

	model.Open()
	require.Zero(t, model.lastListHeight)

	model.Update(intents.BookmarkViewNavigate{Delta: 1, IsPage: true})
	assert.Equal(t, 8, model.listRenderer.GetScrollOffset(), "page navigation should fall back safely before the next render")
}

func TestWindowResize_ResetsCachedPageHeightBeforeNextRender(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(bookmarkListOutput(20)))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.Open())
	model.ViewRect(render.NewDisplayContext(), layout.NewBox(layout.Rect(0, 0, 40, 10)))
	require.Equal(t, 6, model.lastListHeight)

	model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	require.Zero(t, model.lastListHeight)

	model.Update(intents.BookmarkViewNavigate{Delta: 1, IsPage: true})
	assert.Equal(t, 8, model.listRenderer.GetScrollOffset(), "page navigation should not use stale rendered height after resize-before-render")
}

func TestPushSelected_RunsGitPushForBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	commandRunner.Expect(jj.GitPush("--bookmark", "main"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	model.Update(intents.BookmarkViewPush{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestFetchSelected_RunsGitFetchForBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	commandRunner.Expect(jj.GitFetch("--branch", "main"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	model.Update(intents.BookmarkViewFetch{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestPushSelected_RemoteBookmarkUsesRemote(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nmain;origin;true;true;false;false;abc123\n"))
	commandRunner.Expect(jj.GitPush("--bookmark", "main", "--remote", "origin"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())
	model.Update(intents.BookmarkViewToggleExpand{})
	model.Update(intents.BookmarkViewNavigate{Delta: 1})

	model.Update(intents.BookmarkViewPush{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestFetchSelected_RemoteBookmarkUsesRemote(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nmain;origin;true;true;false;false;abc123\n"))
	commandRunner.Expect(jj.GitFetch("--branch", "main", "--remote", "origin"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())
	model.Update(intents.BookmarkViewToggleExpand{})
	model.Update(intents.BookmarkViewNavigate{Delta: 1})

	model.Update(intents.BookmarkViewFetch{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestToggleExpand_ShowsRemoteChildren(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(
		"feature;.;true;false;false;false;abc123\nfeature;origin;true;true;false;false;abc123\nfeature;upstream;true;false;false;false;abc123\n",
	))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())
	require.Len(t, model.visibleRows, 1)

	model.Update(intents.BookmarkViewToggleExpand{})
	require.Len(t, model.visibleRows, 3)

	model.Update(intents.BookmarkViewNavigate{Delta: 1})
	target, ok := model.selectedTarget()
	require.True(t, ok)
	assert.Equal(t, "feature@origin", target)
}

func TestOpen_DeletedLocalBookmarkShowsDeletedState(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(
		"main;.;false;false;false;false;\nmain;origin;true;true;false;false;abc123\n",
	))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("abc123")
	model.SetVisibleCommitIDs([]string{"abc123"})
	test.SimulateModel(model, model.Open())

	require.Len(t, model.visibleRows, 1)
	row := model.visibleRows[0]
	assert.Equal(t, "main", row.Node.Target())
	assert.True(t, row.Node.Deleted)
	assert.False(t, model.tree.Items[row.BookmarkIndex].RemoteOnly)

	model.Update(intents.BookmarkViewToggleExpand{})
	require.Len(t, model.visibleRows, 2)
	assert.Equal(t, "main@origin", model.visibleRows[1].Node.Target())

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 80, 10)))
	assert.Contains(t, dl.RenderToString(80, 10), "deleted")
}

func TestRevealInRevisions_ReturnsMessage(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	cmd := model.Update(intents.BookmarkViewRevealInRevisions{})
	require.NotNil(t, cmd)
	msg, ok := cmd().(ShowBookmarkInRevisionsMsg)
	require.True(t, ok)
	assert.Equal(t, "main", msg.Target)
	assert.Equal(t, "abc123", msg.CommitID)
}

func TestDeleteSelected_UsesSelectedBookmarksForBatchDelete(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeature;.;true;false;false;false;def456\n"))
	commandRunner.Expect(jj.BookmarkDelete("main"))
	commandRunner.Expect(jj.BookmarkDelete("feature"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	model.Update(intents.BookmarkViewToggleSelect{})
	model.Update(intents.BookmarkViewNavigate{Delta: 1})
	model.Update(intents.BookmarkViewToggleSelect{})

	model.Update(intents.BookmarkViewDelete{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestForgetSelected_UsesSelectedBookmarksForBatchForget(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeature;.;true;false;false;false;def456\n"))
	commandRunner.Expect(jj.BookmarkForget("main"))
	commandRunner.Expect(jj.BookmarkForget("feature"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	model.Update(intents.BookmarkViewToggleSelect{})
	model.Update(intents.BookmarkViewNavigate{Delta: 1})
	model.Update(intents.BookmarkViewToggleSelect{})

	model.Update(intents.BookmarkViewForget{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestTrackSelected_UsesSelectedBookmarksForBatchTrack(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;origin;true;false;false;false;abc123\nfeature;upstream;true;false;false;false;def456\n"))
	commandRunner.Expect(jj.BookmarkTrack("main", "origin"))
	commandRunner.Expect(jj.BookmarkTrack("feature", "upstream"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	model.Update(intents.BookmarkViewToggleSelect{})
	model.Update(intents.BookmarkViewNavigate{Delta: 1})
	model.Update(intents.BookmarkViewToggleSelect{})

	model.Update(intents.BookmarkViewTrack{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestUntrackSelected_UsesSelectedBookmarksForBatchUntrack(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;origin;true;true;false;false;abc123\nfeature;upstream;true;true;false;false;def456\n"))
	commandRunner.Expect(jj.BookmarkUntrack("main", "origin"))
	commandRunner.Expect(jj.BookmarkUntrack("feature", "upstream"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	model.Update(intents.BookmarkViewToggleSelect{})
	model.Update(intents.BookmarkViewNavigate{Delta: 1})
	model.Update(intents.BookmarkViewToggleSelect{})

	model.Update(intents.BookmarkViewUntrack{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestMoveSelected_RemoteOnlyBookmarkShowsMessage(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("remote-only;origin;true;true;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	cmd := model.Update(intents.BookmarkViewMove{})
	require.NotNil(t, cmd)
	msg, ok := cmd().(intents.AddMessage)
	require.True(t, ok)
	assert.Equal(t, "No local bookmark for remote-only", msg.Text)
}

func TestEditSelected_RunsAfterConfirmation(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	commandRunner.Expect(jj.Edit("main", false))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	model.Update(intents.BookmarkViewEdit{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestNewFromSelected_RunsAfterConfirmation(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	commandRunner.Expect(jj.New(jj.NewSelectedRevisions(&jj.Commit{ChangeId: "main"})))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	model.Update(intents.BookmarkViewNew{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestConfirmationRendersJustBelowAnchoredBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeature;.;true;false;false;false;def456\n"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())
	anchored, ok := model.selectedTarget()
	require.True(t, ok)

	model.Update(intents.BookmarkViewDelete{})
	require.NotNil(t, model.confirmation)

	rendered := test.RenderImmediate(model, 80, 14)
	lines := strings.Split(rendered, "\n")

	rowLine := findLineContaining(lines, anchored)
	messageLine := findLineContaining(lines, "Are you sure you want to delete the selected bookmark?")
	require.NotEqual(t, -1, rowLine)
	require.NotEqual(t, -1, messageLine)
	assert.LessOrEqual(t, messageLine-rowLine, 2, "confirmation should render directly below the anchored bookmark")
}

func TestConfirmationIgnoresRowClicksWhileOpen(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeature;.;true;false;false;false;def456\n"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())
	anchored, ok := model.selectedTarget()
	require.True(t, ok)
	initialCursor := model.cursor
	clickedIndex := 0
	if initialCursor == 0 {
		clickedIndex = 1
	}

	model.Update(intents.BookmarkViewDelete{})
	require.NotNil(t, model.confirmation)
	assert.Equal(t, anchored, model.confirmationAnchor)

	model.Update(itemClickedMsg{index: clickedIndex})

	assert.Equal(t, initialCursor, model.cursor)
	assert.Equal(t, anchored, model.confirmationAnchor)
}

func findLineContaining(lines []string, fragment string) int {
	for i, line := range lines {
		if strings.Contains(line, fragment) {
			return i
		}
	}
	return -1
}
