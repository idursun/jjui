package revisions

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/operations/workspace"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shaped like `jj log --color=debug` output for builtin_log_compact.
const textObjectLog = "<<log commit node mutable::○>>  " +
	"<<log commit mutable change_id shortest prefix::m>><<log commit mutable change_id shortest rest::mypmnqu>><<log commit mutable:: >>" +
	"<<log commit mutable author email local::someone>><<log commit mutable author email::@>><<log commit mutable author email domain::example.com>><<log commit mutable:: >>" +
	"<<log commit mutable committer timestamp local format::2025-05-22 21:59:38>><<log commit mutable:: >>" +
	"<<log commit mutable bookmarks name::second>><<log commit mutable bookmarks::?? >>" +
	"<<log commit mutable bookmarks name::second>><<log commit mutable bookmarks::@>><<log commit mutable bookmarks remote::origin>><<log commit mutable:: >>" +
	"<<log commit mutable commit_id shortest prefix::28>><<log commit mutable commit_id shortest rest::c8ae4a>><<log commit mutable::>>\n" +
	"│  <<log commit mutable description first_line::second>><<log commit mutable::>>\n" +
	"<<log commit node immutable::◆>>  <<log commit immutable change_id shortest prefix::z>><<log commit immutable change_id shortest rest::zzzzzzz>><<log commit immutable:: >>" +
	"<<log commit immutable root::root()>><<log commit immutable:: >><<log commit immutable commit_id shortest prefix::0>><<log commit immutable commit_id shortest rest::0000000>><<log commit immutable::>>\n"

func textObjectRows(t *testing.T) []parser.Row {
	rows := parser.ParseRows(strings.NewReader(textObjectLog))
	assert.Len(t, rows, 2)
	return rows
}

func TestTextObjectsForRow(t *testing.T) {
	objects := textObjectsForRow(textObjectRows(t)[0])

	type summary struct {
		Kind, Value, Remote string
		Index, line, x, w   int
		Conflicted          bool
	}
	var got []summary
	for _, o := range objects {
		got = append(got, summary{o.Kind, o.Value, o.Remote, o.Index, o.line, o.x, o.width, o.Conflicted})
	}
	assert.Equal(t, []summary{
		{Kind: "node", Value: "○", line: 0, x: 0, w: 1},
		{Kind: "change_id", Value: "mmypmnqu", line: 0, x: 3, w: 8},
		{Kind: "author", Value: "someone@example.com", line: 0, x: 12, w: 19},
		{Kind: "timestamp", Value: "2025-05-22 21:59:38", line: 0, x: 32, w: 19},
		{Kind: "bookmark", Value: "second", line: 0, x: 52, w: 8, Conflicted: true},
		{Kind: "bookmark", Value: "second", Remote: "origin", Index: 1, line: 0, x: 61, w: 13},
		{Kind: "commit_id", Value: "28c8ae4a", line: 0, x: 75, w: 8},
		{Kind: "description", Value: "second", line: 1, x: 3, w: 6},
	}, got)
}

func TestResolveTextObjectFallsBackToSameKindThenNode(t *testing.T) {
	rows := textObjectRows(t)
	first := textObjectsForRow(rows[0])
	assert.Equal(t, "origin", resolveTextObject(first, textObjectBookmark, 1).Remote)
	assert.Equal(t, "second", resolveTextObject(first, textObjectBookmark, 5).Value)

	root := textObjectsForRow(rows[1])
	assert.Equal(t, textObjectNode, resolveTextObject(root, textObjectBookmark, 0).Kind)
}

func TestModel_ObjectFocusIsStickyAcrossRowsAndSetsScope(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)

	assert.Equal(t, textObjectNode, model.focusedTextObject().Kind)
	model.Update(intents.RevisionsObjectFocusKind{Kind: textObjectBookmark})
	assert.Equal(t, textObjectBookmark, model.focusedTextObject().Kind)
	assert.True(t, model.InNormalMode(), "focusing a kind does not open its panel")
	assert.Equal(t, actions.ScopeRevisions+".bookmark", string(model.Scopes()[0].Name))

	model.cursor = 1
	assert.Equal(t, textObjectNode, model.focusedTextObject().Kind, "root has no bookmark, so the node is focused")
	assert.Equal(t, actions.ScopeRevisions, string(model.Scopes()[0].Name))

	model.cursor = 0
	assert.Equal(t, textObjectBookmark, model.focusedTextObject().Kind, "focus returns to the bookmark")

	model.Update(intents.RevisionsObjectNode{})
	assert.Equal(t, textObjectNode, model.focusedTextObject().Kind)
}

func TestModel_MovingObjectFocusStaysInNormalMode(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind = textObjectAuthor

	model.Update(intents.RevisionsObjectFocus{Delta: 1})
	assert.True(t, model.InNormalMode())
	assert.Equal(t, textObjectTimestamp, model.focusedTextObject().Kind)
	scopes := model.Scopes()
	assert.Equal(t, actions.ScopeRevisions+".timestamp", string(scopes[0].Name))
	assert.Equal(t, scopeObjects, scopes[len(scopes)-1].Name)
	assert.True(t, scopes[len(scopes)-1].Global, "h and l stay reachable under details")
}

func TestModel_ObjectFocusWrapsAroundTheRow(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)

	model.Update(intents.RevisionsObjectFocus{Delta: -1})
	assert.Equal(t, textObjectDescription, model.focusedTextObject().Kind)

	model.Update(intents.RevisionsObjectFocus{Delta: 1})
	assert.Equal(t, textObjectNode, model.focusedTextObject().Kind)
	assert.Equal(t, actions.ScopeRevisions, string(model.Scopes()[0].Name))
}

func TestModel_FocusedBookmarkDelete(t *testing.T) {
	for _, tc := range []struct {
		name     string
		index    int
		listing  string
		commands []jj.CommandArgs
	}{
		{name: "local bookmark", index: 0, commands: []jj.CommandArgs{jj.BookmarkDelete("second")}},
		{
			name: "tracked remote bookmark is marked deleted", index: 1,
			listing:  "second;.;true;false;false;false;28\nsecond;origin;true;true;false;false;28\n",
			commands: []jj.CommandArgs{jj.BookmarkListAll(), jj.BookmarkDelete("second")},
		},
		{
			name: "untracked remote bookmark is tracked first", index: 1,
			listing:  "second;origin;true;false;false;false;28\n",
			commands: []jj.CommandArgs{jj.BookmarkListAll(), jj.BookmarkTrack("second", "origin"), jj.BookmarkDelete("second")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			commandRunner := test.NewTestCommandRunner(t)
			for _, command := range tc.commands {
				expectation := commandRunner.Expect(command)
				if command[0] == "bookmark" && command[1] == "list" {
					expectation.SetOutput([]byte(tc.listing))
				}
			}
			defer commandRunner.Verify()

			model := New(test.NewTestContext(commandRunner))
			model.updateGraphRows(textObjectRows(t), "", true)
			model.focusedObjectKind, model.focusedObjectIndex = textObjectBookmark, tc.index
			runAllJJCommands(model.Update(intents.FocusedBookmarkDelete{}))
		})
	}
}

func TestModel_FocusedAuthorResetUpdatesSelectedRevision(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.MetaeditUpdateAuthor("m"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)

	runJJCommand(model.Update(intents.FocusedAuthorReset{}))
	model.focusedObjectKind = textObjectAuthor
	runJJCommand(model.Update(intents.FocusedAuthorReset{}))
}

// runJJCommand runs only the jj command of a RunCommand batch, skipping the
// refresh that would reload the log.
func runJJCommand(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if batch, ok := cmd().(tea.BatchMsg); ok {
		batch[0]()
	}
}

func TestModel_RenameFocusedBookmarkStartsInlineRename(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind = textObjectBookmark

	model.Update(intents.FocusedBookmarkRename{})
	_, isRename := model.baseOperation().(*bookmark.RenameBookmarkOperation)
	assert.True(t, isRename)
	assert.True(t, model.IsEditing())

	model.resetOperations()
	model.focusedObjectIndex = 1
	_, isMessage := model.Update(intents.FocusedBookmarkRename{})().(intents.AddMessage)
	assert.True(t, isMessage, "remote bookmarks cannot be renamed")
}

func TestModel_TrackFocusedBookmarkUsesTrackingState(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	listing := "second;.;true;false;true;false;\nsecond;origin;true;true;false;false;28\n"
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte(listing))
	commandRunner.Expect(jj.BookmarkUntrack("second", "origin"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind, model.focusedObjectIndex = textObjectBookmark, 1

	_, isMessage := model.Update(intents.FocusedBookmarkTrack{Track: true})().(intents.AddMessage)
	assert.True(t, isMessage, "second@origin is already tracked")
	runJJCommand(model.Update(intents.FocusedBookmarkTrack{Track: false}))
}

func TestModel_PushFocusedBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitPush("--bookmark", jj.ExactStringPattern("second")))
	commandRunner.Expect(jj.GitPush("--bookmark", jj.ExactStringPattern("second"), "--remote", "origin"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind = textObjectBookmark
	runJJCommand(model.Update(intents.FocusedBookmarkPush{}))

	model.focusedObjectIndex = 1
	runJJCommand(model.Update(intents.FocusedBookmarkPush{}))
}

func TestTextObjectsForRow_DetailedTemplateBookmarks(t *testing.T) {
	f, err := os.Open("../../../test/testdata/detailed.log")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	rows := parser.ParseRows(f)
	require.Len(t, rows, 2)

	var bookmarks []string
	for _, o := range textObjectsForRow(rows[1]) {
		if o.Kind == textObjectBookmark {
			bookmarks = append(bookmarks, textObjectTitleForTest(o))
		}
	}
	// The detailed template labels them local_bookmarks and remote_bookmarks.
	assert.Equal(t, []string{"main", "main@git"}, bookmarks)
}

func textObjectTitleForTest(o textObject) string {
	if o.Remote != "" {
		return o.Value + "@" + o.Remote
	}
	return o.Value
}

func TestModel_TrackFocusedLocalBookmarkUsesDefaultRemote(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("upstream https://example.com/upstream.git\norigin https://example.com/origin.git\n"))
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("second;.;true;false;true;false;\n"))
	commandRunner.Expect(jj.BookmarkTrack("second", "origin"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind, model.focusedObjectIndex = textObjectBookmark, 0
	runJJCommand(model.Update(intents.FocusedBookmarkTrack{Track: true}))
}

func TestModel_UntrackFocusedLocalBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList()).SetOutput([]byte("origin https://example.com/origin.git\n"))
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("second;.;true;false;false;false;28\nsecond;origin;true;true;false;false;28\n"))
	commandRunner.Expect(jj.BookmarkUntrack("second", "origin"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind, model.focusedObjectIndex = textObjectBookmark, 0

	_, isMessage := model.Update(intents.FocusedBookmarkTrack{Track: true})().(intents.AddMessage)
	assert.True(t, isMessage, "second@origin is already tracked")
	runJJCommand(model.Update(intents.FocusedBookmarkTrack{Track: false}))
}

func TestModel_TrackFocusedLocalBookmarkWithoutRemotes(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitRemoteList())
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind, model.focusedObjectIndex = textObjectBookmark, 0

	msg, isMessage := model.Update(intents.FocusedBookmarkTrack{Track: true})().(intents.AddMessage)
	assert.True(t, isMessage)
	assert.Contains(t, msg.Text, "no git remote")
}

// runAllJJCommands runs every command of nested RunCommand batches; the
// refresh continuations only return a message.
func runAllJJCommands(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range batch {
			runAllJJCommands(c)
		}
	}
}

// Tags as `jj log --color=debug` labels them: a local tag, a local tag that
// moved away from its remote (*) and the remote tag left behind.
const tagLog = "<<log commit node immutable::◆>>  <<log commit immutable change_id shortest prefix::m>><<log commit immutable:: >>" +
	"<<log commit immutable tags name::v1.0>><<log commit immutable tags:: >><<log commit immutable tags name::v2.0>><<log commit immutable tags::*>><<log commit immutable tags:: >>" +
	"<<log commit immutable tags name::v2.0>><<log commit immutable tags::@>><<log commit immutable tags remote::origin>><<log commit immutable:: >>" +
	"<<log commit immutable commit_id shortest prefix::4>><<log commit immutable::>>\n"

func TestTextObjectsForRow_Tags(t *testing.T) {
	rows := parser.ParseRows(strings.NewReader(tagLog))
	require.Len(t, rows, 1)

	var tags []string
	for _, o := range textObjectsForRow(rows[0]) {
		if o.Kind == textObjectTag {
			tags = append(tags, textObjectTitleForTest(o))
		}
	}
	assert.Equal(t, []string{"v1.0", "v2.0", "v2.0@origin"}, tags)
}

func TestModel_FocusedTagDeleteOnlyDeletesLocalTags(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.TagDelete("v1.0"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(parser.ParseRows(strings.NewReader(tagLog)), "", true)
	model.focusedObjectKind = textObjectTag
	assert.Equal(t, actions.ScopeRevisions+".tag", string(model.Scopes()[0].Name))
	runJJCommand(model.Update(intents.FocusedTagDelete{}))

	model.focusedObjectIndex = 2
	_, isMessage := model.Update(intents.FocusedTagDelete{})().(intents.AddMessage)
	assert.True(t, isMessage, "remote tags cannot be deleted")
}

func TestModel_FocusedDescriptionEditDescribesOnlyTheFocusedRevision(t *testing.T) {
	rows := textObjectRows(t)
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Describe(jj.NewSelectedRevisions(rows[0].Commit)))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(rows, "", true)
	// A checked revision would otherwise be described too.
	model.toggleChecked(checkedRevision(rows[1].Commit))
	model.focusedObjectKind = textObjectDescription
	runJJCommand(model.Update(intents.FocusedDescriptionEdit{}))
}

func TestModel_FocusedBookmarkForget(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("second;.;true;false;false;false;28\n"))
	commandRunner.Expect(jj.BookmarkForget("second"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind = textObjectBookmark
	runJJCommand(model.Update(intents.FocusedBookmarkForget{}))
}

func TestModel_FocusedBookmarkForgetIncludesRemotesOfTrackedBookmarks(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListAll()).SetOutput([]byte("second;.;true;false;false;false;28\nsecond;origin;true;true;false;false;28\n"))
	commandRunner.Expect(jj.BookmarkForgetIncludingRemotes("second"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind = textObjectBookmark
	runJJCommand(model.Update(intents.FocusedBookmarkForget{}))
}

func TestModel_FocusedRemoteBookmarkForgetIncludesRemotes(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkForgetIncludingRemotes("second"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind = textObjectBookmark
	model.focusedObjectIndex = 1
	runJJCommand(model.Update(intents.FocusedBookmarkForget{}))
}

func TestModel_ObjectFocusKind(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectIndex = 1
	_, handled := model.HandleIntent(intents.RevisionsObjectFocusKind{Kind: textObjectBookmark})
	assert.True(t, handled)
	assert.Equal(t, 0, model.focusedObjectIndex)
	assert.Equal(t, actions.ScopeRevisions+".bookmark", string(model.Scopes()[0].Name))
}

func TestTextObjectsForRow_UnquotesRefNames(t *testing.T) {
	log := `<<log commit node working_copy mutable::@>>  <<log commit change_id shortest prefix::y>> <<log commit bookmarks name::"a b">><<log commit bookmarks::@>><<log commit bookmarks remote::"my remote">> <<log commit commit_id shortest prefix::3>>` + "\n"
	rows := parser.ParseRows(strings.NewReader(log))
	require.Len(t, rows, 1)
	obj := resolveTextObject(textObjectsForRow(rows[0]), textObjectBookmark, 0)
	require.NotNil(t, obj)
	assert.Equal(t, "a b", obj.Value)
	assert.Equal(t, "my remote", obj.Remote)
}

const divergentLog = "<<log commit node immutable::◆>>  <<log commit immutable divergent change_id shortest prefix::nkmpu>><<log commit immutable divergent change_id shortest rest::xny>><<log commit immutable divergent change_offset::/2>><<log commit immutable:: >>" +
	"<<log commit immutable author email local::yc>><<log commit immutable:: >>" +
	"<<log commit immutable commit_id shortest prefix::7fa9>><<log commit immutable commit_id shortest rest::215d>><<log commit immutable:: >><<log commit immutable divergent::(divergent)>><<log commit immutable::>>\n"

func divergentRows(t *testing.T) []parser.Row {
	rows := parser.ParseRows(strings.NewReader(divergentLog))
	require.Len(t, rows, 1)
	return rows
}

func TestTextObjectsForRow_Divergent(t *testing.T) {
	objects := textObjectsForRow(divergentRows(t)[0])
	var kinds, values []string
	for _, o := range objects {
		kinds = append(kinds, o.Kind)
		values = append(values, o.Value)
	}
	assert.Equal(t, []string{"node", "change_id", "author", "commit_id", "divergent"}, kinds)
	assert.Equal(t, []string{"◆", "nkmpuxny/2", "yc", "7fa9215d", "(divergent)"}, values)
}

func TestModel_FocusedDivergentActions(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.AbandonCommit("7fa9"))
	commandRunner.Expect(jj.MetaeditUpdateChangeId("7fa9"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(divergentRows(t), "", true)
	model.focusedObjectKind = textObjectDivergent
	runJJCommand(model.Update(intents.FocusedDivergentAbandon{}))
	runJJCommand(model.Update(intents.FocusedDivergentUpdateChangeId{}))

	cmd, _ := model.HandleIntent(intents.FocusedDivergentShowAll{})
	assert.Equal(t, common.UpdateRevSetMsg("change_id(nkmpu)"), cmd())
}

// Two workspaces share the working-copy commit and render in one segment.
const workspaceLog = "<<log commit node working_copy mutable::@>>  " +
	"<<log commit working_copy mutable change_id shortest prefix::s>><<log commit working_copy mutable change_id shortest rest::mqwonot>><<log commit working_copy mutable:: >>" +
	"<<log commit working_copy mutable working_copies::default@ second@>><<log commit working_copy mutable:: >>" +
	"<<log commit working_copy mutable commit_id shortest prefix::05>><<log commit working_copy mutable commit_id shortest rest::0bef07>><<log commit working_copy mutable::>>\n"

func workspaceModel(t *testing.T, commandRunner *test.CommandRunner, location string) *Model {
	ctx := test.NewTestContext(commandRunner)
	ctx.Location = location
	model := New(ctx)
	model.updateGraphRows(parser.ParseRows(strings.NewReader(workspaceLog)), "", true)
	model.focusedObjectKind, model.focusedObjectIndex = textObjectWorkspace, 1
	return model
}

// Shaped like `jj log --color=debug` output for a conflicted row.
const conflictLog = "<<log commit node mutable conflicted::×>>  " +
	"<<log commit mutable conflicted change_id shortest prefix::wpz>><<log commit mutable conflicted change_id shortest rest::zoqpo>><<log commit mutable conflicted:: >>" +
	"<<log commit mutable conflicted commit_id shortest prefix::fc>><<log commit mutable conflicted commit_id shortest rest::efc5f9>><<log commit mutable conflicted:: >>" +
	"<<log commit mutable conflicted conflict::(conflict)>><<log commit mutable conflicted::>>\n"

func TestTextObjectsForRow_Conflict(t *testing.T) {
	rows := parser.ParseRows(strings.NewReader(conflictLog))
	require.Len(t, rows, 1)

	var kinds, values []string
	var conflicts []string
	for _, o := range textObjectsForRow(rows[0]) {
		kinds = append(kinds, o.Kind)
		values = append(values, o.Value)
		if o.Kind == textObjectConflict {
			conflicts = append(conflicts, o.Value)
		}
	}
	assert.Equal(t, []string{"(conflict)"}, conflicts)
	assert.Equal(t, []string{"node", "change_id", "commit_id", "conflict"}, kinds)
	assert.Equal(t, []string{"×", "wpzzoqpo", "fcefc5f9", "(conflict)"}, values)
}

func TestTextObjectsForRow_Workspaces(t *testing.T) {
	rows := parser.ParseRows(strings.NewReader(workspaceLog))
	require.Len(t, rows, 1)

	type summary struct {
		Value    string
		Index, x int
		w        int
	}
	var got []summary
	for _, o := range textObjectsForRow(rows[0]) {
		if o.Kind == textObjectWorkspace {
			got = append(got, summary{o.Value, o.Index, o.x, o.width})
		}
	}
	assert.Equal(t, []summary{{"default", 0, 12, 8}, {"second", 1, 21, 7}}, got)
}

func TestModel_FocusedWorkspaceSwitchChangesLocation(t *testing.T) {
	other := t.TempDir()
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.WorkspaceRoot("second")).SetOutput([]byte(other + "\n"))
	defer commandRunner.Verify()

	model := workspaceModel(t, commandRunner, "/repo")
	assert.Equal(t, actions.ScopeRevisions+".workspace", string(model.Scopes()[0].Name))
	model.Update(intents.FocusedWorkspaceSwitch{})
	assert.Equal(t, other, model.context.Location)

	_, isMessage := model.Update(intents.FocusedWorkspaceSwitch{})().(intents.AddMessage)
	assert.True(t, isMessage, "switching to the current workspace does nothing")
}

func TestModel_FocusedWorkspaceForget(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.WorkspaceRoot("second")).SetOutput([]byte("/second"))
	commandRunner.Expect(jj.WorkspaceForget("second"))
	defer commandRunner.Verify()

	model := workspaceModel(t, commandRunner, "/repo")
	runJJCommand(model.Update(intents.FocusedWorkspaceForget{}))

	model.context.Location = "/second"
	_, isMessage := model.Update(intents.FocusedWorkspaceForget{})().(intents.AddMessage)
	assert.True(t, isMessage, "the current workspace cannot be forgotten")
}

func TestModel_FocusedWorkspaceRenameOnlyRenamesTheCurrentWorkspace(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.WorkspaceRoot("second")).SetOutput([]byte("/second"))
	defer commandRunner.Verify()

	model := workspaceModel(t, commandRunner, "/repo")
	_, isMessage := model.Update(intents.FocusedWorkspaceRename{})().(intents.AddMessage)
	assert.True(t, isMessage, "other workspaces cannot be renamed")

	model.context.Location = "/second"
	model.Update(intents.FocusedWorkspaceRename{})
	_, isRename := model.baseOperation().(*workspace.RenameWorkspaceOperation)
	assert.True(t, isRename)
	assert.True(t, model.IsEditing())
}

func jumpLogRow(changeId, bookmark string) string {
	row := "<<log commit node mutable::○>>  <<log commit mutable change_id shortest prefix::" + changeId + ">><<log commit mutable:: >>"
	if bookmark != "" {
		row += "<<log commit mutable bookmarks name::" + bookmark + ">><<log commit mutable:: >>"
	}
	return row + "<<log commit mutable commit_id shortest prefix::" + changeId + "0>><<log commit mutable::>>\n"
}

func jumpRows(t *testing.T, log string) []parser.Row {
	return parser.ParseRows(strings.NewReader(log))
}

func TestModel_ObjectJumpMovesToTheNextRevisionWithTheSameKind(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(jumpRows(t, jumpLogRow("a", "one")+jumpLogRow("b", "")+jumpLogRow("c", "two")), "", true)
	model.focusedObjectKind = textObjectBookmark
	assert.Equal(t, "one", model.focusedTextObject().Value)

	model.Update(intents.RevisionsObjectJump{Delta: 1})
	assert.Equal(t, 2, model.cursor)
	assert.Equal(t, "two", model.focusedTextObject().Value)

	_, isMessage := model.Update(intents.RevisionsObjectJump{Delta: 1})().(intents.AddMessage)
	assert.True(t, isMessage, "there is no bookmark below the last one")
	assert.Equal(t, 2, model.cursor)

	model.Update(intents.RevisionsObjectJump{Delta: -1})
	assert.Equal(t, 0, model.cursor)
	assert.Equal(t, "one", model.focusedTextObject().Value)
}

func TestModel_ObjectJumpContinuesInTheNextBatch(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	first := jumpRows(t, jumpLogRow("a", "one")+jumpLogRow("b", ""))
	model.updateGraphRows(first, "", true)
	model.offScreenRows = first
	model.hasMore = true
	model.focusedObjectKind = textObjectBookmark

	model.Update(intents.RevisionsObjectJump{Delta: 1})
	assert.Equal(t, 0, model.cursor, "the cursor waits for the next batch")
	assert.Equal(t, textObjectBookmark, model.pendingObjectJump)

	model.Update(appendRowsBatchMsg{rows: jumpRows(t, jumpLogRow("c", "two")), tag: model.tag.Load()})
	assert.Equal(t, 2, model.cursor)
	assert.Equal(t, "two", model.focusedTextObject().Value)
	assert.Equal(t, "", model.pendingObjectJump)
}

func TestModel_ClickingAnObjectFocusesIt(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.cursor = 1

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 120, 10)))

	// The remote bookmark second@origin starts at column 61 of the first row.
	msg, handled := dl.ProcessMouseEvent(tea.MouseClickMsg{X: 63, Y: 0, Button: tea.MouseLeft})
	require.True(t, handled)
	assert.Equal(t, ObjectClickedMsg{Index: 0, Kind: textObjectBookmark, ObjectIndex: 1}, msg)

	model.Update(msg)
	assert.Equal(t, 0, model.cursor)
	obj := model.focusedTextObject()
	assert.Equal(t, textObjectBookmark, obj.Kind)
	assert.Equal(t, "origin", obj.Remote)

	msg, _ = dl.ProcessMouseEvent(tea.MouseClickMsg{X: 63, Y: 0, Button: tea.MouseLeft, Mod: tea.ModCtrl})
	assert.Equal(t, ItemClickedMsg{Index: 0, Ctrl: true}, msg, "ctrl-click still checks the revision")
}

func TestModel_CheckingARevisionDoesNotMoveItsObjects(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.Update(intents.RevisionsToggleSelect{})
	model.cursor = 0
	model.focusedObjectKind = textObjectBookmark

	dl := render.NewDisplayContext()
	model.ViewRect(dl, layout.NewBox(layout.Rect(0, 0, 120, 10)))

	cursor := dl.Cursor()
	require.NotNil(t, cursor)
	assert.Equal(t, 52, cursor.X, "the bookmark starts at column 52 of the jj output")

	msg, _ := dl.ProcessMouseEvent(tea.MouseClickMsg{X: 50, Y: 0, Button: tea.MouseLeft})
	assert.Equal(t, ObjectClickedMsg{Index: 0, Kind: textObjectTimestamp}, msg)
	msg, _ = dl.ProcessMouseEvent(tea.MouseClickMsg{X: 52, Y: 0, Button: tea.MouseLeft})
	assert.Equal(t, ObjectClickedMsg{Index: 0, Kind: textObjectBookmark}, msg)
}

func TestModel_FocusedBookmarkAdvanceOnlyAdvancesLocalBookmarks(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkAdvance("second"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind = textObjectBookmark
	runJJCommand(model.Update(intents.FocusedBookmarkAdvance{}))

	model.focusedObjectIndex = 1
	_, isMessage := model.Update(intents.FocusedBookmarkAdvance{})().(intents.AddMessage)
	assert.True(t, isMessage, "remote bookmarks cannot be advanced")
}

func checkBookmarksOnTwoRevisions(t *testing.T, commandRunner *test.CommandRunner) *Model {
	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(jumpRows(t, jumpLogRow("a", "one")+jumpLogRow("b", "two")), "", true)
	model.focusedObjectKind = textObjectBookmark
	model.Update(intents.RevisionsToggleSelect{})
	model.cursor = 1
	model.Update(intents.RevisionsToggleSelect{})
	return model
}

func TestModel_ToggleSelectChecksTheFocusedBookmarkNotItsRevision(t *testing.T) {
	model := checkBookmarksOnTwoRevisions(t, test.NewTestCommandRunner(t))

	assert.Equal(t, []checkedObject{checkedBookmark("one", ""), checkedBookmark("two", "")}, model.checked)

	model.Update(intents.RevisionsToggleSelect{})
	assert.Equal(t, []checkedObject{checkedBookmark("one", "")}, model.checked)
}

func TestModel_RevisionActionsResolveCheckedBookmarksToTheirRevisions(t *testing.T) {
	model := checkBookmarksOnTwoRevisions(t, test.NewTestCommandRunner(t))

	var changeIds []string
	for _, commit := range model.SelectedRevisions().Revisions {
		changeIds = append(changeIds, commit.GetChangeId())
	}
	assert.Equal(t, []string{model.rows[0].Commit.GetChangeId(), model.rows[1].Commit.GetChangeId()}, changeIds)
	assert.Len(t, model.Selection().Checked, 2)
}

func TestModel_BookmarkDeleteAppliesToCheckedBookmarks(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkDelete("one", "two"))
	defer commandRunner.Verify()

	runAllJJCommands(checkBookmarksOnTwoRevisions(t, commandRunner).Update(intents.FocusedBookmarkDelete{}))
}

func TestModel_BookmarkActionsPreferCheckedOverFocused(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkDelete("one"))
	defer commandRunner.Verify()

	model := checkBookmarksOnTwoRevisions(t, commandRunner)
	model.Update(intents.RevisionsToggleSelect{}) // unchecks "two", the cursor stays on it
	runAllJJCommands(model.Update(intents.FocusedBookmarkDelete{}))
}

func TestModel_BookmarkPushRunsOncePerRemote(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitPush("--bookmark", jj.ExactStringPattern("second")))
	commandRunner.Expect(jj.GitPush("--bookmark", jj.ExactStringPattern("second"), "--remote", "origin"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind = textObjectBookmark
	model.Update(intents.RevisionsToggleSelect{})
	model.focusedObjectIndex = 1
	model.Update(intents.RevisionsToggleSelect{})
	runAllJJCommands(model.Update(intents.FocusedBookmarkPush{}))
}

func TestModel_RefreshClearsCheckedBookmarksUnlessKept(t *testing.T) {
	model := checkBookmarksOnTwoRevisions(t, test.NewTestCommandRunner(t))
	model.refresh(intents.Refresh{KeepSelections: true})
	assert.Len(t, model.checked, 2)
	model.refresh(intents.Refresh{})
	assert.Empty(t, model.checked)
}

func TestModel_BookmarkAdvanceAppliesToCheckedLocalBookmarks(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkAdvance("one", "two"))
	defer commandRunner.Verify()

	runAllJJCommands(checkBookmarksOnTwoRevisions(t, commandRunner).Update(intents.FocusedBookmarkAdvance{}))
}

func TestModel_BookmarkAdvanceSkipsCheckedRemoteBookmarks(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkAdvance("second"))
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind = textObjectBookmark
	model.Update(intents.RevisionsToggleSelect{})
	model.focusedObjectIndex = 1
	model.Update(intents.RevisionsToggleSelect{})
	runAllJJCommands(model.Update(intents.FocusedBookmarkAdvance{}))
}

func TestModel_BookmarkMoveAppliesToCheckedBookmarks(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkMove("a", []string{"one", "two"}))
	defer commandRunner.Verify()

	model := checkBookmarksOnTwoRevisions(t, commandRunner)
	model.Update(intents.FocusedBookmarkMove{})
	operation, ok := model.baseOperation().(*bookmark.MoveBookmarkOperation)
	require.True(t, ok)
	operation.SetSelectedRevision(model.rows[0].Commit)
	runAllJJCommands(operation.Update(intents.Apply{}))
}

func TestModel_BookmarkMoveOnlyRemoteBookmarksShowsMessage(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind, model.focusedObjectIndex = textObjectBookmark, 1

	_, isMessage := model.Update(intents.FocusedBookmarkMove{})().(intents.AddMessage)
	assert.True(t, isMessage)
}

func TestModel_CheckedTargetsKeepCheckOrderAcrossKinds(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(jumpRows(t, jumpLogRow("a", "one")+jumpLogRow("b", "two")), "", true)
	model.toggleChecked(checkedBookmark("two", ""))
	model.toggleChecked(checkedRevision(model.rows[0].Commit))

	commits := model.checkedTargetCommits()
	require.Len(t, commits, 2)
	assert.Equal(t, model.rows[1].Commit.CommitId, commits[0].CommitId)
	assert.Equal(t, model.rows[0].Commit.CommitId, commits[1].CommitId)
	assert.Equal(t, 2, model.CheckedCount())
}

func checkDescriptionsAndAuthorsOnTwoRevisions(t *testing.T, commandRunner *test.CommandRunner) *Model {
	description := func(text string) string {
		return "│  <<log commit mutable description first_line::" + text + ">><<log commit mutable::>>\n"
	}
	author := "<<log commit mutable author email local::me>><<log commit mutable:: >>"
	header := func(changeId string) string {
		return "<<log commit node mutable::○>>  <<log commit mutable change_id shortest prefix::" + changeId + ">><<log commit mutable:: >>" +
			author + "<<log commit mutable commit_id shortest prefix::" + changeId + "0>><<log commit mutable::>>\n"
	}
	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(jumpRows(t, header("a")+description("x")+header("b")+description("y")), "", true)
	for _, kind := range []string{textObjectDescription, textObjectAuthor} {
		model.focusedObjectKind, model.focusedObjectIndex = kind, 0
		for cursor := range model.rows {
			model.cursor = cursor
			model.Update(intents.RevisionsToggleSelect{})
		}
	}
	return model
}

func TestModel_DescriptionEditAppliesToCheckedDescriptions(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	model := checkDescriptionsAndAuthorsOnTwoRevisions(t, commandRunner)
	commandRunner.Expect(jj.Describe(jj.NewSelectedRevisions(model.rows[0].Commit, model.rows[1].Commit)))
	defer commandRunner.Verify()

	model.focusedObjectKind = textObjectDescription
	runJJCommand(model.Update(intents.FocusedDescriptionEdit{}))
}

func TestModel_AuthorResetAppliesToCheckedAuthors(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.MetaeditUpdateAuthor("a", "b"))
	defer commandRunner.Verify()

	runJJCommand(checkDescriptionsAndAuthorsOnTwoRevisions(t, commandRunner).Update(intents.FocusedAuthorReset{}))
}

func TestModel_RevisionActionsResolveCheckedDescriptions(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind = textObjectDescription
	model.Update(intents.RevisionsToggleSelect{})

	assert.Equal(t, []checkedObject{{kind: textObjectDescription, commitId: model.rows[0].Commit.CommitId}}, model.checked)
	require.Len(t, model.SelectedRevisions().Revisions, 1)
}

func changeIdFocusedUnderDetails(t *testing.T, commandRunner *test.CommandRunner) *Model {
	t.Helper()
	ctx := test.NewTestContext(commandRunner)
	model := New(ctx)
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind = textObjectChangeId
	commit := model.SelectedRevision()
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(commit.GetChangeId())).SetOutput([]byte("false false $\nM file.txt\n"))
	op := details.NewOperation(ctx, commit, model.fileChecks())
	test.SimulateModel(op, op.Init())
	model.baseOp = op
	return model
}

func scopeNames(model *Model) []string {
	var names []string
	for _, scope := range model.Scopes() {
		names = append(names, string(scope.Name))
	}
	return names
}

func TestModel_ChangeIdScopeIsVisibleUnderDetailsOnlyOnTheChangeId(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()
	model := changeIdFocusedUnderDetails(t, commandRunner)
	assert.Contains(t, scopeNames(model), actions.ScopeRevisions+".change_id")

	model.baseOp.(*details.Operation).HandleIntent(intents.DetailsNavigate{Delta: 1})
	assert.NotContains(t, scopeNames(model), actions.ScopeRevisions+".change_id")
}

func TestModel_PushFocusedChange(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	model := changeIdFocusedUnderDetails(t, commandRunner)
	commandRunner.Expect(jj.GitPush("--change", model.SelectedRevision().GetChangeId()))
	defer commandRunner.Verify()

	runJJCommand(model.Update(intents.FocusedChangePush{}))
}

func TestModel_PushChangesPushesCheckedRevisionsInOneCommand(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	first, second := model.rows[0].Commit, model.rows[1].Commit
	commandRunner.Expect(jj.GitPush("--change", first.GetChangeId(), "--change", second.GetChangeId()))
	defer commandRunner.Verify()

	model.toggleChecked(checkedRevision(first))
	model.toggleChecked(checkedRevision(second))
	model.focusedObjectKind = textObjectChangeId
	runJJCommand(model.Update(intents.FocusedChangePush{}))
}

func TestModel_ObjectScopeSitsBeforeRevisionsOffTheNode(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)

	names := func() []string {
		var ret []string
		for _, s := range model.Scopes() {
			ret = append(ret, string(s.Name))
		}
		return ret
	}

	model.Update(intents.RevisionsObjectFocusKind{Kind: textObjectBookmark})
	got := names()
	assert.Equal(t, []string{"revisions.bookmark", "revisions.object", "revisions"}, got[:3])

	model.Update(intents.RevisionsObjectNode{})
	assert.NotContains(t, names(), "revisions.object")
}

func checkFileUnderDetails(t *testing.T, model *Model) {
	t.Helper()
	op := model.baseOp.(*details.Operation)
	op.HandleIntent(intents.DetailsNavigate{Delta: 1})
	op.HandleIntent(intents.DetailsToggleSelect{})
}

func TestModel_CheckedFileIsNotARevisionTarget(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()
	model := changeIdFocusedUnderDetails(t, commandRunner)
	checkFileUnderDetails(t, model)

	require.Len(t, model.checked, 1)
	assert.Equal(t, textObjectFile, model.checked[0].kind)
	assert.Equal(t, model.SelectedRevision().GetChangeId(), model.checked[0].changeId)
	assert.Equal(t, "file.txt", model.checked[0].name)
	assert.Empty(t, model.checkedTargetCommits())
	assert.Empty(t, model.checkedRevisionItems())
	require.Len(t, model.SelectedRevisions().Revisions, 1)
	assert.Equal(t, model.SelectedRevision().CommitId, model.SelectedRevisions().Revisions[0].CommitId)
	hasChecked, _ := model.QueryState("has_checked")
	assert.Equal(t, false, hasChecked)
}

func TestModel_CheckedFileIsCountedAndListedOnce(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()
	model := changeIdFocusedUnderDetails(t, commandRunner)
	checkFileUnderDetails(t, model)
	model.toggleChecked(checkedRevision(model.rows[1].Commit))

	assert.Equal(t, 2, model.CheckedCount())
	var files, revisions int
	for _, item := range model.Selection().Checked {
		switch item.(type) {
		case appContext.SelectedFile:
			files++
		case appContext.SelectedRevision:
			revisions++
		}
	}
	assert.Equal(t, 1, files)
	assert.Equal(t, 1, revisions)
}

func TestModel_FileChecksBelongToTheirRevision(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()
	model := changeIdFocusedUnderDetails(t, commandRunner)
	model.toggleChecked(checkedFile("another-change", "file.txt"))

	assert.Empty(t, model.baseOp.(*details.Operation).Selection().Checked)
	assert.Equal(t, 0, model.CheckedCount())
}

func TestModel_FileChecksSurviveRefreshWhileDetailsStayOpen(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	defer commandRunner.Verify()
	model := changeIdFocusedUnderDetails(t, commandRunner)
	checkFileUnderDetails(t, model)
	model.toggleChecked(checkedBookmark("second", ""))

	model.refresh(intents.Refresh{})

	require.Len(t, model.checked, 1)
	assert.Equal(t, textObjectFile, model.checked[0].kind)
}

func TestModel_FileChecksAreDroppedWhenDetailsClose(t *testing.T) {
	for name, closeDetails := range map[string]func(*Model){
		"reset operations":              func(m *Model) { m.resetOperations() },
		"sync details":                  func(m *Model) { m.focusedObjectKind = textObjectNode; m.syncDetails() },
		"replaced by another operation": func(m *Model) { m.setBaseOperation(operations.NewDefault()) },
	} {
		t.Run(name, func(t *testing.T) {
			commandRunner := test.NewTestCommandRunner(t)
			model := changeIdFocusedUnderDetails(t, commandRunner)
			checkFileUnderDetails(t, model)
			model.toggleChecked(checkedBookmark("second", ""))
			require.Len(t, model.checked, 2)

			closeDetails(model)

			assert.Equal(t, []checkedObject{checkedBookmark("second", "")}, model.checked)
		})
	}
}

func TestModel_ObjectCycleKindFocusesFirstThenCyclesWithWrapAround(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)

	_, handled := model.HandleIntent(intents.RevisionsObjectCycleKind{Kind: textObjectBookmark})
	assert.True(t, handled)
	assert.Equal(t, textObjectBookmark, model.focusedObjectKind)
	assert.Equal(t, 0, model.focusedObjectIndex)

	model.HandleIntent(intents.RevisionsObjectCycleKind{Kind: textObjectBookmark})
	assert.Equal(t, 1, model.focusedObjectIndex)

	model.HandleIntent(intents.RevisionsObjectCycleKind{Kind: textObjectBookmark})
	assert.Equal(t, 0, model.focusedObjectIndex)
}

func TestModel_ObjectCycleKindStartsAtFirstWhenAnotherKindIsFocused(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind, model.focusedObjectIndex = textObjectAuthor, 0

	model.HandleIntent(intents.RevisionsObjectCycleKind{Kind: textObjectBookmark})
	assert.Equal(t, textObjectBookmark, model.focusedObjectKind)
	assert.Equal(t, 0, model.focusedObjectIndex)
}

func TestModel_ObjectCycleKindWithoutMatchKeepsFocusAndReportsIt(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)
	model.focusedObjectKind, model.focusedObjectIndex = textObjectAuthor, 0

	cmd, handled := model.HandleIntent(intents.RevisionsObjectCycleKind{Kind: textObjectTag})
	assert.True(t, handled)
	assert.Equal(t, textObjectAuthor, model.focusedObjectKind)
	require.NotNil(t, cmd)
	invoke, ok := cmd().(intents.Intent)
	require.True(t, ok)
	assert.Equal(t, intents.AddMessage{Text: "No tag on this revision"}, invoke)
}

func TestModel_ObjectFocusKindDoesNotCycle(t *testing.T) {
	model := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	model.updateGraphRows(textObjectRows(t), "", true)

	model.HandleIntent(intents.RevisionsObjectFocusKind{Kind: textObjectBookmark})
	model.HandleIntent(intents.RevisionsObjectFocusKind{Kind: textObjectBookmark})
	assert.Equal(t, 0, model.focusedObjectIndex)
}
