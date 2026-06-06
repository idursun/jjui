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

func TestOnShow_SortsLocalBookmarksFirstByDistanceAndSelectsClosestMoveable(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(
		"remote-only;origin;true;true;false;false;ccc333\ncurrent;.;true;false;false;false;bbb222\nfar-local;.;true;false;false;false;ddd444\nnear-local;.;true;false;false;false;aaa111\n",
	))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("bbb222", []string{"aaa111", "bbb222", "ccc333", "ddd444"})
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	require.Len(t, model.visibleRows, 4)

	row, ok := model.rowNode(model.visibleRows[0])
	require.True(t, ok)
	assert.Equal(t, "current", row.Target())
	row, ok = model.rowNode(model.visibleRows[1])
	require.True(t, ok)
	assert.Equal(t, "far-local", row.Target())
	row, ok = model.rowNode(model.visibleRows[2])
	require.True(t, ok)
	assert.Equal(t, "near-local", row.Target())
	row, ok = model.rowNode(model.visibleRows[3])
	require.True(t, ok)
	assert.Equal(t, "remote-only@origin", row.Target())

	target, ok := model.selectedTarget()
	require.True(t, ok)
	assert.Equal(t, "current", target)
	assert.Equal(t, "bbb222", model.selectedCommitID())
}

func TestSyncRevisionContext_ResortsAndPreservesSelectedBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(
		"remote-only;origin;true;true;false;false;ccc333\ncurrent;.;true;false;false;false;bbb222\nfar-local;.;true;false;false;false;ddd444\nnear-local;.;true;false;false;false;aaa111\n",
	))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	visibleCommitIDs := []string{"aaa111", "bbb222", "ccc333", "ddd444"}
	model.SyncRevisionContext("bbb222", visibleCommitIDs)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)
	model.moveCursor(1)
	model.listRenderer.StartLine = 2

	target, ok := model.selectedTarget()
	require.True(t, ok)
	require.Equal(t, "far-local", target)

	model.SyncRevisionContext("aaa111", visibleCommitIDs)

	require.Len(t, model.visibleRows, 4)
	first, ok := model.rowNode(model.visibleRows[0])
	require.True(t, ok)
	assert.Equal(t, "near-local", first.Target())
	target, ok = model.selectedTarget()
	require.True(t, ok)
	assert.Equal(t, "far-local", target)
	assert.Equal(t, 2, model.listRenderer.StartLine)
}

func TestRenameSelected_LocalBookmarkOpensPrompt(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	cmd := model.Update(intents.BookmarkPaneRename{})
	require.NotNil(t, cmd)
	showInput, ok := cmd().(common.ShowInputMsg)
	require.True(t, ok, "rename should request input")
	assert.Equal(t, "main", showInput.Value)
}

func TestCreateSelected_ReturnsBeginCreateMessage(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	cmd := model.Update(intents.BookmarkPaneCreate{})
	require.NotNil(t, cmd)
	msg, ok := cmd().(BeginCreateBookmarkMsg)
	require.True(t, ok)
	assert.Equal(t, BeginCreateBookmarkMsg{}, msg)
}

func TestCreateSelected_ReturnsBeginCreateMessageWhenNoBookmarksVisible(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(""))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	cmd := model.Update(intents.BookmarkPaneCreate{})
	require.NotNil(t, cmd)
	msg, ok := cmd().(BeginCreateBookmarkMsg)
	require.True(t, ok)
	assert.Equal(t, BeginCreateBookmarkMsg{}, msg)
}

func TestRevealSelected_ReturnsRevealMessageWithCommitID(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", []string{"abc123"})
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	cmd := model.Update(intents.BookmarkPaneShowInRevision{})
	require.NotNil(t, cmd)
	msg, ok := cmd().(RevealRevisionMsg)
	require.True(t, ok)
	assert.Equal(t, "abc123", msg.CommitID)
}

func TestRevealSelected_WhenAlreadyAtBookmark_ReturnsRevealMessage(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("abc123", []string{"abc123"})
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	cmd := model.Update(intents.BookmarkPaneShowInRevision{})
	require.NotNil(t, cmd)
	msg, ok := cmd().(RevealRevisionMsg)
	require.True(t, ok)
	assert.Equal(t, "abc123", msg.CommitID)
}

func TestBookmarkPanePageNavigation_UsesRenderedListHeight(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(bookmarkListOutput(20)))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 10)))
	require.Equal(t, 7, model.lastListHeight, "list height should reflect rendered viewport height")
	require.Equal(t, 0, model.listRenderer.GetScrollOffset())
	require.Equal(t, 0, model.listRenderer.GetFirstRowIndex())

	model.Update(intents.BookmarkPaneNavigate{Delta: 1, IsPage: true})
	assert.Equal(t, 7, model.listRenderer.GetScrollOffset())

	dl = render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 40, 10)))
	assert.Equal(t, 7, model.listRenderer.GetFirstRowIndex())
	assert.Equal(t, 13, model.listRenderer.GetLastRowIndex())
}

func TestOpen_ResetsCachedPageHeightBeforeFirstRender(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(bookmarkListOutput(20)))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)
	model.ViewRect(render.NewDisplayContext(), layout.NewBox(layout.Rect(0, 0, 40, 10)))
	require.Equal(t, 7, model.lastListHeight)

	model.OnHide()
	require.Zero(t, model.lastListHeight)

	model.OnShow()
	model.SetFocused(true)
	require.Zero(t, model.lastListHeight)

	model.Update(intents.BookmarkPaneNavigate{Delta: 1, IsPage: true})
	assert.Equal(t, 8, model.listRenderer.GetScrollOffset(), "page navigation should fall back safely before the next render")
}

func TestWindowResize_ResetsCachedPageHeightBeforeNextRender(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(bookmarkListOutput(20)))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)
	model.ViewRect(render.NewDisplayContext(), layout.NewBox(layout.Rect(0, 0, 40, 10)))
	require.Equal(t, 7, model.lastListHeight)

	model.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	require.Zero(t, model.lastListHeight)

	model.Update(intents.BookmarkPaneNavigate{Delta: 1, IsPage: true})
	assert.Equal(t, 8, model.listRenderer.GetScrollOffset(), "page navigation should not use stale rendered height after resize-before-render")
}

func TestPushSelected_RunsGitPushForBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	commandRunner.Expect(jj.GitPush("--bookmark", "main"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	model.Update(intents.BookmarkPanePush{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestFetchSelected_RunsGitFetchForBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	commandRunner.Expect(jj.GitFetch("--branch", "main"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	model.Update(intents.BookmarkPaneFetch{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestPushSelected_RemoteBookmarkUsesRemote(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nmain;origin;true;true;false;false;abc123\n"))
	commandRunner.Expect(jj.GitPush("--bookmark", "main", "--remote", "origin"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)
	model.Update(intents.BookmarkPaneToggleExpand{})
	model.Update(intents.BookmarkPaneNavigate{Delta: 1})

	model.Update(intents.BookmarkPanePush{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestFetchSelected_RemoteBookmarkUsesRemote(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nmain;origin;true;true;false;false;abc123\n"))
	commandRunner.Expect(jj.GitFetch("--branch", "main", "--remote", "origin"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)
	model.Update(intents.BookmarkPaneToggleExpand{})
	model.Update(intents.BookmarkPaneNavigate{Delta: 1})

	model.Update(intents.BookmarkPaneFetch{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestCycleRemotes_FiltersRowsBySelectedRemote(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(
		"main;.;true;false;false;false;abc123\nmain;origin;true;true;false;false;abc123\nremote-only;origin;true;true;false;false;def456\nupstream-only;upstream;true;true;false;false;fed321\n",
	))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	assert.Equal(t, []string{allRemoteFilter, localRemoteFilter, "origin", "upstream"}, model.remoteNames)
	require.Len(t, model.visibleRows, 3)

	model.Update(intents.BookmarkPaneCycleRemotes{Delta: 1})
	assert.Equal(t, localRemoteFilter, model.activeRemoteFilter())
	require.Len(t, model.visibleRows, 1)
	target, ok := model.selectedTarget()
	require.True(t, ok)
	assert.Equal(t, "main", target)

	model.Update(intents.BookmarkPaneCycleRemotes{Delta: 1})
	assert.Equal(t, "origin", model.activeRemoteFilter())
	require.Len(t, model.visibleRows, 2)
	originTargets := make([]string, 0, len(model.visibleRows))
	for _, row := range model.visibleRows {
		node, ok := model.rowNode(row)
		require.True(t, ok)
		originTargets = append(originTargets, node.Target())
	}
	assert.Equal(t, []string{"main@origin", "remote-only@origin"}, originTargets)

	model.Update(intents.BookmarkPaneCycleRemotes{Delta: -1})
	assert.Equal(t, localRemoteFilter, model.activeRemoteFilter())
}

func TestToggleExpand_ShowsRemoteChildren(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(
		"feature;.;true;false;false;false;abc123\nfeature;origin;true;true;false;false;abc123\nfeature;upstream;true;false;false;false;abc123\n",
	))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)
	require.Len(t, model.visibleRows, 1)

	model.Update(intents.BookmarkPaneToggleExpand{})
	require.Len(t, model.visibleRows, 3)

	model.Update(intents.BookmarkPaneNavigate{Delta: 1})
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

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("abc123", []string{"abc123"})
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	require.Len(t, model.visibleRows, 1)
	row := model.visibleRows[0]
	node, ok := model.rowNode(row)
	require.True(t, ok)
	assert.Equal(t, "main", node.Target())
	assert.True(t, node.Deleted)
	assert.False(t, model.tree.Items[row.BookmarkIndex].remoteOnly())

	model.Update(intents.BookmarkPaneToggleExpand{})
	require.Len(t, model.visibleRows, 2)
	node, ok = model.rowNode(model.visibleRows[1])
	require.True(t, ok)
	assert.Equal(t, "main@origin", node.Target())

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 80, 10)))
	assert.Contains(t, dl.RenderToString(80, 10), "deleted")
}

func TestRevealInRevisions_ReturnsUpdateRevset(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	cmd := model.Update(intents.BookmarkPaneSetRevset{})
	require.NotNil(t, cmd)
	msg, ok := cmd().(common.UpdateRevSetMsg)
	require.True(t, ok)
	assert.Equal(t, common.UpdateRevSetMsg("::main"), msg)
}

func TestCancel_ClearsSelectionsBeforeClosingPane(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeature;.;true;false;false;false;def456\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	model.Update(intents.BookmarkPaneToggleSelect{})
	require.Len(t, model.selected, 1)

	cmd, handled := model.HandleIntent(intents.Cancel{})
	require.True(t, handled)
	assert.Nil(t, cmd)
	assert.Empty(t, model.selected)
	assert.True(t, model.Focused())

	cmd, handled = model.HandleIntent(intents.Cancel{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	_, ok := cmd().(common.CloseViewMsg)
	assert.True(t, ok)
}

func TestCloseAndOpen_ClearSelectionsFromPreviousSession(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeature;.;true;false;false;false;def456\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)
	model.Update(intents.BookmarkPaneToggleSelect{})
	require.Len(t, model.selected, 1)

	model.OnHide()
	assert.Empty(t, model.selected)

	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)
	assert.Empty(t, model.selected)
}

func TestRowsLoaded_PrunesSelectionsThatNoLongerExist(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeature;.;true;false;false;false;def456\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	selectedTarget, ok := model.selectedTarget()
	require.True(t, ok)
	model.Update(intents.BookmarkPaneToggleSelect{})
	require.Equal(t, map[string]bool{selectedTarget: true}, model.selected)

	remainingOutput := "main;.;true;false;false;false;abc123\n"
	if selectedTarget == "main" {
		remainingOutput = "feature;.;true;false;false;false;def456\n"
	}
	model.Update(rowsLoadedMsg{tree: loadBookmarkTree(remainingOutput, model.expanded, "", nil)})
	assert.Empty(t, model.selected)
}

func TestDeleteSelected_UsesSelectedBookmarksForBatchDelete(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeature;.;true;false;false;false;def456\n"))
	commandRunner.Expect(jj.BookmarkDelete("main"))
	commandRunner.Expect(jj.BookmarkDelete("feature"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	model.Update(intents.BookmarkPaneToggleSelect{})
	model.Update(intents.BookmarkPaneNavigate{Delta: 1})
	model.Update(intents.BookmarkPaneToggleSelect{})

	model.Update(intents.BookmarkPaneDelete{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestForgetSelected_UsesSelectedBookmarksForBatchForget(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeature;.;true;false;false;false;def456\n"))
	commandRunner.Expect(jj.BookmarkForget("main"))
	commandRunner.Expect(jj.BookmarkForget("feature"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	model.Update(intents.BookmarkPaneToggleSelect{})
	model.Update(intents.BookmarkPaneNavigate{Delta: 1})
	model.Update(intents.BookmarkPaneToggleSelect{})

	model.Update(intents.BookmarkPaneForget{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestForgetSelected_RemoteOnlyBookmarkRunsForget(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("remote-only;origin;true;true;false;false;abc123\n"))
	commandRunner.Expect(jj.BookmarkForget("remote-only"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	model.Update(intents.BookmarkPaneForget{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestTrackSelected_UsesSelectedBookmarksForBatchTrack(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;origin;true;false;false;false;abc123\nfeature;upstream;true;false;false;false;def456\n"))
	commandRunner.Expect(jj.BookmarkTrack("main", "origin"))
	commandRunner.Expect(jj.BookmarkTrack("feature", "upstream"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	model.Update(intents.BookmarkPaneToggleSelect{})
	model.Update(intents.BookmarkPaneNavigate{Delta: 1})
	model.Update(intents.BookmarkPaneToggleSelect{})

	model.Update(intents.BookmarkPaneTrack{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestUntrackSelected_UsesSelectedBookmarksForBatchUntrack(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;origin;true;true;false;false;abc123\nfeature;upstream;true;true;false;false;def456\n"))
	commandRunner.Expect(jj.BookmarkUntrack("main", "origin"))
	commandRunner.Expect(jj.BookmarkUntrack("feature", "upstream"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	model.Update(intents.BookmarkPaneToggleSelect{})
	model.Update(intents.BookmarkPaneNavigate{Delta: 1})
	model.Update(intents.BookmarkPaneToggleSelect{})

	model.Update(intents.BookmarkPaneUntrack{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestUntrackSelected_LocalBookmarkUntracksAllTrackedRemotes(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nmain;origin;true;true;false;false;abc123\nmain;upstream;true;true;false;false;abc123\n"))
	commandRunner.Expect(jj.BookmarkUntrack("main", "origin"))
	commandRunner.Expect(jj.BookmarkUntrack("main", "upstream"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	model.Update(intents.BookmarkPaneUntrack{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestDeleteSelected_DeletedLocalBookmarkDoesNothing(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;false;false;false;false;\nmain;origin;true;true;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	cmd := model.Update(intents.BookmarkPaneDelete{})
	assert.Nil(t, cmd)
	assert.Nil(t, model.confirmation)
}

func TestDeleteSelected_RemoteRowDoesNotDeleteLocalBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nmain;origin;true;true;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)
	model.Update(intents.BookmarkPaneToggleExpand{})
	model.Update(intents.BookmarkPaneNavigate{Delta: 1})

	target, ok := model.selectedTarget()
	require.True(t, ok)
	require.Equal(t, "main@origin", target)

	cmd := model.Update(intents.BookmarkPaneDelete{})
	assert.Nil(t, cmd)
	assert.Nil(t, model.confirmation)
}

func TestMoveSelected_RemoteOnlyBookmarkShowsMessage(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("remote-only;origin;true;true;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	cmd := model.Update(intents.BookmarkPaneMove{})
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

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	model.Update(intents.BookmarkPaneEdit{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestNewFromSelected_RunsAfterConfirmation(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\n"))
	commandRunner.Expect(jj.New(jj.NewSelectedRevisions(&jj.Commit{ChangeId: "abc123"})))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	model.Update(intents.BookmarkPaneNew{})
	require.NotNil(t, model.confirmation)
	test.SimulateModel(model, func() tea.Msg { return confirmation.SelectOptionMsg{Index: 0} })
}

func TestConfirmationRendersAsModal(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeature;.;true;false;false;false;def456\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)

	model.Update(intents.BookmarkPaneDelete{})
	require.NotNil(t, model.confirmation)

	dl := render.NewDisplayContext()
	box := layout.NewBox(layout.Rect(0, 0, 80, 14))
	model.ViewRect(dl, box)
	model.RenderOverlay(dl, box)
	screen := render.NewScreenBuffer(80, 14)
	dl.Render(screen)
	rendered := screen.Render()
	assert.Contains(t, rendered, "Are you sure you want to delete the selected bookmark?")
	assert.Contains(t, rendered, "Yes")
	assert.Contains(t, rendered, "No")
}

func TestConfirmationIgnoresRowClicksWhileOpen(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeature;.;true;false;false;false;def456\n"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.SyncRevisionContext("dest123", nil)
	test.SimulateModel(model, model.OnShow())
	model.SetFocused(true)
	initialCursor := model.cursor
	clickedIndex := 0
	if initialCursor == 0 {
		clickedIndex = 1
	}

	model.Update(intents.BookmarkPaneDelete{})
	require.NotNil(t, model.confirmation)

	model.Update(ItemClickedMsg{Index: clickedIndex})

	assert.Equal(t, initialCursor, model.cursor)
}
