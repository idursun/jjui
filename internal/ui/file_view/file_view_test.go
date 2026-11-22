package file_view

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var commit = &jj.Commit{ChangeId: "abcdefghijk", CommitId: "c0ffee"}

// annotated is a line as jj's default templates.file_annotate prints it
// under `--color=debug`.
func annotated(change, number, content string) string {
	return "<<commit change_id shortest prefix::" + change + ">> <<commit author email local::me>>" +
		"    <<line_number::" + number + ">>: <<content::" + content + ">>\n"
}

// fixture is a three line file: lines 1-2 from change one, line 3 from two.
var fixture = annotated("one", "1", "package a") +
	annotated("one", "2", "x := y >> 2") +
	annotated("two", "3", "")

func TestParseAnnotationReadsTheLabels(t *testing.T) {
	lines := parseAnnotation([]byte(fixture))

	require.Len(t, lines, 3)
	assert.Equal(t, line{changeId: "one", number: 1, content: "package a", gutter: "one me    1: ", first: true}, lines[0])
	assert.Equal(t, line{changeId: "one", number: 2, content: "x := y >> 2", gutter: strings.Repeat(" ", len("one me    ")) + "2: "}, lines[1])
	assert.Equal(t, line{changeId: "two", number: 3, gutter: "two me    3: ", first: true}, lines[2])
}

func TestParseAnnotationReadsACommitIdTemplate(t *testing.T) {
	lines := parseAnnotation([]byte("<<commit commit_id shortest prefix::c>><<commit commit_id shortest rest::1>> <<content::x>>\n"))

	require.Len(t, lines, 1)
	assert.Equal(t, "c1", lines[0].id())
}

func TestTemplateWithoutIdsDoesNotJumpOrGoOlder(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), commit, jj.NewFileName("a.go"))
	m.Update(annotationLoadedMsg{frame: m.current(), lines: parseAnnotation([]byte("<<line_number::1>>: <<content::x>>\n"))})

	jump, _ := m.HandleIntent(intents.FileViewJumpToChange{})
	older, _ := m.HandleIntent(intents.FileViewOlder{})

	assert.Nil(t, jump)
	assert.Nil(t, older)
	assert.Len(t, m.stack, 1)
}

func openOn(t *testing.T, revset string) (*Model, *test.CommandRunner) {
	runner := test.NewTestCommandRunner(t)
	ctx := test.NewTestContext(runner)
	ctx.DefaultRevset, ctx.CurrentRevset = "builtin_log()", revset
	m := New(ctx, commit, jj.NewFileName("a.go"))
	m.Update(annotationLoadedMsg{frame: m.current(), lines: parseAnnotation([]byte(fixture))})
	return m, runner
}

func run(m *Model, cmd tea.Cmd) {
	if cmd != nil {
		m.Update(cmd())
	}
}

func TestOpeningWithoutAPathStartsOnThePathLine(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.FileAnnotate("c0ffee", jj.NewFileName("src/main.go"))).SetOutput([]byte(fixture))
	m := New(test.NewTestContext(runner), commit, jj.FileName{})
	m.Update(filesLoadedMsg{paths: []string{"README.md", "src/main.go"}})

	assert.True(t, m.IsEditing())
	assert.Equal(t, "file_view.path", string(m.Scopes()[0].Name))
	for _, r := range "main" {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	require.Len(t, m.matches, 1)

	cmd, _ := m.HandleIntent(intents.FileViewPathAccept{})
	run(m, cmd)

	assert.False(t, m.IsEditing())
	assert.Equal(t, "src/main.go", m.path.Path())
	assert.Len(t, m.current().lines, 3)
	runner.Verify()
}

func TestCancellingThePathLineWithoutAFileCloses(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), commit, jj.FileName{})

	cmd, _ := m.HandleIntent(intents.FileViewPathCancel{})

	require.NotNil(t, cmd)
	assert.Equal(t, common.CloseViewMsg{}, cmd())
}

func TestMoveChangeJumpsBetweenHunks(t *testing.T) {
	m, _ := openOn(t, "builtin_log()")

	m.HandleIntent(intents.FileViewMoveChange{Delta: 1})
	assert.Equal(t, 2, m.current().cursor)
	m.HandleIntent(intents.FileViewMoveChange{Delta: 1})
	assert.Equal(t, 2, m.current().cursor)

	m.HandleIntent(intents.FileViewMoveChange{Delta: -1})
	assert.Equal(t, 0, m.current().cursor)
}

func TestOlderAnnotatesBeforeTheChangeAndNewerReturns(t *testing.T) {
	m, runner := openOn(t, "builtin_log()")
	m.HandleIntent(intents.FileViewNavigate{Delta: 1})
	runner.Expect(jj.FileAnnotate("first_parent(one)", jj.NewFileName("a.go"))).SetOutput([]byte(fixture))

	cmd, _ := m.HandleIntent(intents.FileViewOlder{})
	run(m, cmd)

	require.Len(t, m.stack, 2)
	assert.Equal(t, 1, m.current().cursor)
	runner.Verify()

	m.HandleIntent(intents.FileViewNewer{})
	require.Len(t, m.stack, 1)
	assert.Equal(t, 1, m.current().cursor)
}

func TestSelectionIsTheChangeUnderTheCursor(t *testing.T) {
	m, _ := openOn(t, "builtin_log()")
	m.HandleIntent(intents.FileViewMoveChange{Delta: 1})

	assert.Equal(t, common.SelectedFile{ChangeId: "two", File: jj.NewFileName("a.go")}, m.Selection().Highlighted)
}

// The log holds the shortest change id, which can be shorter than the one
// the annotate template prints.
func TestJumpToChangeSelectsAChangeTheRevsetShows(t *testing.T) {
	m, runner := openOn(t, "trunk()..@")
	runner.Expect(jj.GetIdsFromRevset("(trunk()..@) & one")).SetOutput([]byte("on\n"))

	cmd, handled := m.HandleIntent(intents.FileViewJumpToChange{})

	require.True(t, handled)
	var msgs []tea.Msg
	test.SimulateModel(m, cmd, func(msg tea.Msg) { msgs = append(msgs, msg) })
	assert.Equal(t, []tea.Msg{jumpMsg{id: "one", selected: "on", shown: true}, common.CloseViewMsg{}, common.RefreshMsg{SelectedRevision: "on"}}, msgs)
}

func TestJumpToChangeAddsAChangeTheRevsetHides(t *testing.T) {
	m, runner := openOn(t, "trunk()..@")
	runner.Expect(jj.GetIdsFromRevset("(trunk()..@) & one"))
	runner.Expect(jj.GetIdsFromRevset("one")).SetOutput([]byte("on\n"))

	cmd, _ := m.HandleIntent(intents.FileViewJumpToChange{})

	var msgs []tea.Msg
	test.SimulateModel(m, cmd, func(msg tea.Msg) { msgs = append(msgs, msg) })
	assert.Equal(t, []tea.Msg{
		jumpMsg{id: "one", selected: "on"},
		common.CloseViewMsg{},
		common.UpdateRevSetMsg("(trunk()..@) | one"),
		common.RefreshMsg{SelectedRevision: "on"},
	}, msgs)
}

func TestFilterRevsetNarrowsTheRevset(t *testing.T) {
	tests := []struct{ revset, want string }{
		{"trunk()..@", `trunk()..@ & files(file:"a.go")`},
		{"builtin_log()", `files(file:"a.go")`},
	}
	for _, tt := range tests {
		m, _ := openOn(t, tt.revset)

		cmd, _ := m.HandleIntent(intents.FileViewFilterRevset{})

		msgs := []tea.Msg{}
		for _, c := range cmd().(tea.BatchMsg) {
			msgs = append(msgs, c())
		}
		assert.Contains(t, msgs, common.UpdateRevSetMsg(tt.want))
		assert.Contains(t, msgs, common.CloseViewMsg{})
	}
}

func TestMoveTopAndBottom(t *testing.T) {
	m, _ := openOn(t, "builtin_log()")

	m.HandleIntent(intents.FileViewNavigate{Delta: 1, ToEnd: true})
	assert.Equal(t, 2, m.current().cursor)
	m.HandleIntent(intents.FileViewNavigate{Delta: -1, ToEnd: true})
	assert.Equal(t, 0, m.current().cursor)
}

func TestHalfPageMovesHalfAsFarAsAPage(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	m := New(test.NewTestContext(runner), commit, jj.NewFileName("a.go"))
	var out string
	for i := 1; i <= 100; i++ {
		out += annotated("one", strconv.Itoa(i), "x")
	}
	m.Update(annotationLoadedMsg{frame: m.current(), lines: parseAnnotation([]byte(out))})
	test.RenderImmediate(m, 80, 22)

	m.HandleIntent(intents.FileViewNavigate{Delta: 1, IsPage: true})
	page := m.current().cursor
	m.HandleIntent(intents.FileViewNavigate{Delta: -1, ToEnd: true})
	m.HandleIntent(intents.FileViewNavigate{Delta: 1, IsHalfPage: true})

	require.Greater(t, page, 2)
	assert.Equal(t, page/2, m.current().cursor)
}

func openWithLongLine(t *testing.T) *Model {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), commit, jj.NewFileName("a.txt"))
	content := "start " + strings.Repeat("x", 100) + " end"
	m.Update(annotationLoadedMsg{frame: m.current(), lines: parseAnnotation([]byte(annotated("one", "1", content)))})
	return m
}

func TestScrollRightStopsAtTheEndOfTheWidestLine(t *testing.T) {
	m := openWithLongLine(t)
	test.RenderImmediate(m, 60, 5)

	for range 200 {
		m.HandleIntent(intents.FileViewScrollHorizontal{Delta: 1})
	}
	screen := test.RenderImmediate(m, 60, 5)

	assert.Contains(t, screen, "x end")
	assert.NotContains(t, screen, "start")
	assert.Equal(t, 110-m.contentWidth, m.scrollX)
}

func TestWrapShowsTheWholeLine(t *testing.T) {
	m := openWithLongLine(t)
	m.HandleIntent(intents.FileViewScrollHorizontal{Delta: 5})

	m.HandleIntent(intents.FileViewToggleWrap{})
	screen := test.RenderImmediate(m, 60, 5)

	assert.Contains(t, screen, "start")
	assert.Contains(t, screen, "x end")
	assert.Equal(t, 0, m.scrollX)
}

func openOnLines(t *testing.T, contents ...string) *Model {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), commit, jj.NewFileName("a.txt"))
	var out string
	for i, content := range contents {
		out += annotated("one", strconv.Itoa(i+1), content)
	}
	m.Update(annotationLoadedMsg{frame: m.current(), lines: parseAnnotation([]byte(out))})
	return m
}

func search(m *Model, text string) {
	m.HandleIntent(intents.FileViewSearch{})
	for _, r := range text {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m.HandleIntent(intents.FileViewSearchApply{})
}

func TestSearchMovesBetweenMatchesAndWraps(t *testing.T) {
	m := openOnLines(t, "apple", "Banana", "grape", "banana split")

	search(m, "banana")
	assert.Equal(t, 1, m.current().cursor)

	m.HandleIntent(intents.QuickSearchCycle{})
	assert.Equal(t, 3, m.current().cursor)
	m.HandleIntent(intents.QuickSearchCycle{})
	assert.Equal(t, 1, m.current().cursor)
	m.HandleIntent(intents.QuickSearchCycle{Reverse: true})
	assert.Equal(t, 3, m.current().cursor)
}

func TestSearchInputTakesTheKeys(t *testing.T) {
	m := openOnLines(t, "apple")

	m.HandleIntent(intents.FileViewSearch{})

	assert.True(t, m.IsEditing())
	assert.Equal(t, "file_view.search_input", string(m.Scopes()[0].Name))
	m.HandleIntent(intents.FileViewSearchCancel{})
	assert.False(t, m.IsEditing())
	assert.Equal(t, "file_view", string(m.Scopes()[0].Name))
}

func TestEmptySearchRepeatsThePreviousQuery(t *testing.T) {
	m := openOnLines(t, "x", "x", "y")
	search(m, "x")
	m.HandleIntent(intents.FileViewNavigate{Delta: 1, ToEnd: true})

	search(m, "")

	assert.Equal(t, "x", m.query)
	assert.Equal(t, 0, m.current().cursor)
}

func TestCloseClearsTheSearchBeforeClosing(t *testing.T) {
	m := openOnLines(t, "apple")
	search(m, "app")

	cmd, _ := m.HandleIntent(intents.FileViewClose{})
	assert.Nil(t, cmd)
	assert.Empty(t, m.query)

	cmd, _ = m.HandleIntent(intents.FileViewClose{})
	require.NotNil(t, cmd)
	assert.Equal(t, common.CloseViewMsg{}, cmd())
}
