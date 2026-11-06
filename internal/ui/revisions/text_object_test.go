package revisions

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/actions"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
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
	for range 4 {
		model.Update(intents.RevisionsObjectFocus{Delta: 1})
	}
	assert.Equal(t, textObjectBookmark, model.focusedTextObject().Kind)
	assert.Equal(t, actions.ScopeRevisions+".bookmark", string(model.Scopes()[0].Name))

	model.cursor = 1
	assert.Equal(t, textObjectNode, model.focusedTextObject().Kind, "root has no bookmark, so the node is focused")
	assert.Equal(t, actions.ScopeRevisions, string(model.Scopes()[0].Name))

	model.cursor = 0
	assert.Equal(t, textObjectBookmark, model.focusedTextObject().Kind, "focus returns to the bookmark")

	model.Update(intents.RevisionsObjectNode{})
	assert.Equal(t, textObjectNode, model.focusedTextObject().Kind)
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
	model.toggleCheckedRevision(appContext.SelectedRevision{ChangeId: rows[1].Commit.ChangeId, CommitId: rows[1].Commit.CommitId})
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

func TestModel_RevisionsFetch(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GitFetch())
	defer commandRunner.Verify()

	model := New(test.NewTestContext(commandRunner))
	model.updateGraphRows(textObjectRows(t), "", true)
	runJJCommand(model.Update(intents.RevisionsFetch{}))
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
