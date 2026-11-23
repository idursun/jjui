package bookmark

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestSetBookmarkModel_Update(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovable("revision"))
	commandRunner.Expect(jj.BookmarkSet("revision", "name"))
	defer commandRunner.Verify()

	op := NewSetBookmarkOperation(test.NewTestContext(commandRunner), "revision", "", nil)
	test.SimulateModel(op, op.Init())
	test.SimulateModel(op, test.Type("name"))
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func TestSetBookmarkModel_Prefill(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.BookmarkListMovable("revision"))
	commandRunner.Expect(jj.BookmarkSet("revision", "rdeaton/20260425/feature"))
	defer commandRunner.Verify()

	op := NewSetBookmarkOperation(test.NewTestContext(commandRunner), "revision", "rdeaton/20260425/", nil)
	test.SimulateModel(op, op.Init())
	if got := op.name.Value(); got != "rdeaton/20260425/" {
		t.Fatalf("expected prefilled value %q, got %q", "rdeaton/20260425/", got)
	}
	test.SimulateModel(op, test.Type("feature"))
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

// The log, top to bottom: up2, up1, target, down1, down2.
var rankingLog = []*jj.Commit{
	{ChangeId: "up2", CommitId: "aaaa"},
	{ChangeId: "up1", CommitId: "bbbb"},
	{ChangeId: "target", CommitId: "cccc"},
	{ChangeId: "down1", CommitId: "dddd"},
	{ChangeId: "down2", CommitId: "eeee"},
}

// Fields: name;remote;present;tracked;conflict;backwards;commit id.
const rankingBookmarks = "far;.;true;false;false;false;e\n" +
	"near;.;true;false;false;false;d\n" +
	"above-far;.;true;false;false;true;a\n" +
	"above-near;.;true;false;false;true;b\n" +
	"here;.;true;false;false;true;c\n" +
	"elsewhere;.;true;false;false;false;f\n"

func TestRankCandidates_NearestBelowThenAboveThenOutsideTheLog(t *testing.T) {
	candidates := rankCandidates("target", rankingLog, jj.ParseBookmarkListOutput(rankingBookmarks))
	var names []string
	for _, c := range candidates {
		names = append(names, c.name)
	}
	assert.Equal(t, []string{"near", "far", "above-near", "above-far", "elsewhere"}, names)
}

func TestRankCandidates_WithoutTargetInLogSkipsBackwards(t *testing.T) {
	candidates := rankCandidates("target", nil, jj.ParseBookmarkListOutput(rankingBookmarks))
	var names []string
	for _, c := range candidates {
		names = append(names, c.name)
	}
	assert.Equal(t, []string{"elsewhere", "far", "near"}, names)
}

func newRankingOperation(t *testing.T, commandRunner *test.CommandRunner) *SetBookmarkOperation {
	t.Helper()
	commandRunner.Expect(jj.BookmarkListMovable("target")).SetOutput([]byte(rankingBookmarks))
	op := NewSetBookmarkOperation(test.NewTestContext(commandRunner), "target", "", rankingLog)
	test.SimulateModel(op, op.Init())
	return op
}

func TestSetBookmarkModel_EmptyInputMovesNearest(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	op := newRankingOperation(t, commandRunner)
	commandRunner.Expect(jj.BookmarkSet("target", "near"))
	defer commandRunner.Verify()

	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func TestSetBookmarkModel_CycleStepsThroughRankedCandidates(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	op := newRankingOperation(t, commandRunner)
	commandRunner.Expect(jj.BookmarkSet("target", "above-near", "--allow-backwards"))
	defer commandRunner.Verify()

	for range 3 {
		test.SimulateModel(op, func() tea.Msg { return intents.AutocompleteCycle{} })
	}
	assert.Equal(t, "above-near", op.name.Value())
	test.SimulateModel(op, func() tea.Msg { return intents.Apply{} })
}

func TestSetBookmarkModel_PreviewNamesChosenBookmark(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	op := newRankingOperation(t, commandRunner)
	defer commandRunner.Verify()

	assert.Equal(t, " ← near", op.preview())

	test.SimulateModel(op, test.Type("above-far"))
	assert.Equal(t, " backwards", op.preview())

	test.SimulateModel(op, test.Type("-new"))
	assert.Empty(t, op.preview())
}

func TestSetBookmarkModel_RenderSegmentReversesOnlyChosen(t *testing.T) {
	commandRunner := test.NewTestCommandRunner(t)
	op := newRankingOperation(t, commandRunner)
	defer commandRunner.Verify()

	near := &screen.Segment{Text: "near", Labels: []string{"bookmarks", "name"}}
	far := &screen.Segment{Text: "far", Labels: []string{"bookmarks", "name"}}
	row := parser.Row{
		Commit: rankingLog[3],
		Lines:  []*parser.GraphRowLine{{Segments: []*screen.Segment{near, {Text: " "}, far}}},
	}

	assert.NotEmpty(t, op.RenderSegment(lipgloss.NewStyle(), near, row), "chosen bookmark is reversed")
	assert.Empty(t, op.RenderSegment(lipgloss.NewStyle(), far, row), "other bookmarks keep their style")

	test.SimulateModel(op, test.Type("n"))
	assert.Empty(t, op.RenderSegment(lipgloss.NewStyle(), far, row), "non-matching bookmarks keep their style")
}
