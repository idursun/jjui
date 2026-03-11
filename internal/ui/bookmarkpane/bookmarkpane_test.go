package bookmarkpane

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_LoadsRowsAndPreselectsCurrentRevisionBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(
		"main;.;false;false;false;aaa111\nfeature;.;false;false;false;bbb222\nfeature;origin;true;false;false;bbb222\n",
	))
	defer commandRunner.Verify()

	model := NewModel(
		test.NewTestContext(commandRunner),
		func() *jj.Commit { return &jj.Commit{ChangeId: "feature-change", CommitId: "bbb222"} },
		func(string) tea.Cmd { return nil },
	)
	test.SimulateModel(model, model.Open())

	require.True(t, model.Visible())
	target, ok := model.selectedTarget()
	require.True(t, ok)
	assert.Equal(t, "feature", target)
	assert.Equal(t, "bbb222", model.selectedCommitID())
}

func TestRenameSelected_LocalBookmarkOpensPrompt(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	model := NewModel(
		test.NewTestContext(commandRunner),
		func() *jj.Commit { return &jj.Commit{ChangeId: "dest", CommitId: "dest123"} },
		func(string) tea.Cmd { return nil },
	)
	test.SimulateModel(model, model.Open())

	cmd := model.Update(intents.BookmarkViewRename{})
	require.NotNil(t, cmd)
	showInput, ok := cmd().(common.ShowInputMsg)
	require.True(t, ok, "rename should request input")
	assert.Equal(t, "main", showInput.Value)
}

func TestRevealSelected_UsesCallbackWithCommitID(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;false;false;false;abc123\n"))
	defer commandRunner.Verify()

	var revealed string
	model := NewModel(
		test.NewTestContext(commandRunner),
		func() *jj.Commit { return &jj.Commit{ChangeId: "dest", CommitId: "dest123"} },
		func(revision string) tea.Cmd {
			revealed = revision
			return func() tea.Msg { return nil }
		},
	)
	test.SimulateModel(model, model.Open())

	cmd := model.Update(intents.BookmarkViewReveal{})
	require.NotNil(t, cmd)
	_ = cmd()
	assert.Equal(t, "abc123", revealed)
}

func TestToggleExpand_ShowsRemoteChildren(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(
		"feature;.;false;false;false;abc123\nfeature;origin;true;false;false;abc123\nfeature;upstream;false;false;false;abc123\n",
	))
	defer commandRunner.Verify()

	model := NewModel(
		test.NewTestContext(commandRunner),
		func() *jj.Commit { return &jj.Commit{ChangeId: "dest", CommitId: "dest123"} },
		func(string) tea.Cmd { return nil },
	)
	test.SimulateModel(model, model.Open())
	require.Len(t, model.visibleEntries, 1)

	model.Update(intents.BookmarkViewToggleExpand{})
	require.Len(t, model.visibleEntries, 3)

	model.Update(intents.BookmarkViewNavigate{Delta: 1})
	target, ok := model.selectedTarget()
	require.True(t, ok)
	assert.Equal(t, "feature@origin", target)
}
