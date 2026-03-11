package bookmarkpane

import (
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_SortsLocalBookmarksFirstByDistanceAndSelectsClosestMoveable(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(
		"remote-only;origin;true;false;false;ccc333\ncurrent;.;false;false;false;bbb222\nfar-local;.;false;false;false;ddd444\nnear-local;.;false;false;false;aaa111\n",
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
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;false;false;false;abc123\n"))
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

func TestRevealSelected_ReturnsRevealMessageWithCommitID(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;false;false;false;abc123\n"))
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
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;false;false;false;abc123\n"))
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

func TestToggleExpand_ShowsRemoteChildren(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(
		"feature;.;false;false;false;abc123\nfeature;origin;true;false;false;abc123\nfeature;upstream;false;false;false;abc123\n",
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

func TestRevealInRevisions_ReturnsMessage(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;false;false;false;abc123\n"))
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
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;false;false;false;abc123\nfeature;.;false;false;false;def456\n"))
	commandRunner.Expect(jj.BookmarkDelete("main"))
	commandRunner.Expect(jj.BookmarkDelete("feature"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	model.Update(intents.BookmarkViewToggleSelect{})
	model.Update(intents.BookmarkViewNavigate{Delta: 1})
	model.Update(intents.BookmarkViewToggleSelect{})

	cmd := model.Update(intents.BookmarkViewDelete{})
	require.NotNil(t, cmd)
	test.SimulateModel(model, cmd)
}

func TestForgetSelected_UsesSelectedBookmarksForBatchForget(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;false;false;false;abc123\nfeature;.;false;false;false;def456\n"))
	commandRunner.Expect(jj.BookmarkForget("main"))
	commandRunner.Expect(jj.BookmarkForget("feature"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	model.Update(intents.BookmarkViewToggleSelect{})
	model.Update(intents.BookmarkViewNavigate{Delta: 1})
	model.Update(intents.BookmarkViewToggleSelect{})

	cmd := model.Update(intents.BookmarkViewForget{})
	require.NotNil(t, cmd)
	test.SimulateModel(model, cmd)
}

func TestTrackSelected_UsesSelectedBookmarksForBatchTrack(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;origin;false;false;false;abc123\nfeature;upstream;false;false;false;def456\n"))
	commandRunner.Expect(jj.BookmarkTrack("main", "origin"))
	commandRunner.Expect(jj.BookmarkTrack("feature", "upstream"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	model.Update(intents.BookmarkViewToggleSelect{})
	model.Update(intents.BookmarkViewNavigate{Delta: 1})
	model.Update(intents.BookmarkViewToggleSelect{})

	cmd := model.Update(intents.BookmarkViewTrack{})
	require.NotNil(t, cmd)
	test.SimulateModel(model, cmd)
}

func TestUntrackSelected_UsesSelectedBookmarksForBatchUntrack(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;origin;true;false;false;abc123\nfeature;upstream;true;false;false;def456\n"))
	commandRunner.Expect(jj.BookmarkUntrack("main", "origin"))
	commandRunner.Expect(jj.BookmarkUntrack("feature", "upstream"))
	defer commandRunner.Verify()

	model := NewModel(test.NewTestContext(commandRunner))
	model.SetCurrentCommitID("dest123")
	test.SimulateModel(model, model.Open())

	model.Update(intents.BookmarkViewToggleSelect{})
	model.Update(intents.BookmarkViewNavigate{Delta: 1})
	model.Update(intents.BookmarkViewToggleSelect{})

	cmd := model.Update(intents.BookmarkViewUntrack{})
	require.NotNil(t, cmd)
	test.SimulateModel(model, cmd)
}

func TestMoveSelected_RemoteOnlyBookmarkShowsMessage(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("remote-only;origin;true;false;false;abc123\n"))
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
