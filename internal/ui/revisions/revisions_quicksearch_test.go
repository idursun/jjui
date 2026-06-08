package revisions

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

var searchableRows = []parser.Row{
	{
		Commit: &jj.Commit{ChangeId: "first", CommitId: "111"},
		Lines: []*parser.GraphRowLine{
			{
				Gutter:   parser.GraphGutter{Segments: []*screen.Segment{{Text: "|"}}},
				Segments: []*screen.Segment{{Text: "first match"}},
				Flags:    parser.Revision,
			},
		},
	},
	{
		Commit: &jj.Commit{ChangeId: "second", CommitId: "222"},
		Lines: []*parser.GraphRowLine{
			{
				Gutter:   parser.GraphGutter{Segments: []*screen.Segment{{Text: "|"}}},
				Segments: []*screen.Segment{{Text: "second match"}},
				Flags:    parser.Revision,
			},
		},
	},
	{
		Commit: &jj.Commit{ChangeId: "third", CommitId: "333"},
		Lines: []*parser.GraphRowLine{
			{
				Gutter:   parser.GraphGutter{Segments: []*screen.Segment{{Text: "|"}}},
				Segments: []*screen.Segment{{Text: "third match"}},
				Flags:    parser.Revision,
			},
		},
	},
}

// TestQuickSearch_ClearIntentClearsSearch tests that clear intent clears the search.
func TestQuickSearch_ClearIntentClearsSearch(t *testing.T) {
	model := &Model{
		quickSearch: "test",
		baseOp:      operations.NewDefault(),
		rows:        []parser.Row{{Commit: &jj.Commit{ChangeId: "test123"}}},
	}

	cmd := model.internalUpdate(intents.RevisionsQuickSearchClear{})

	assert.Equal(t, "", model.quickSearch, "clear intent should clear quicksearch")
	assert.Nil(t, cmd)
}

func TestQuickSearch_UpdatesSelection(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(searchableRows, "first")

	selectionChanged := func(cmd tea.Cmd) bool {
		var changed bool
		test.SimulateModel(model, cmd, func(msg tea.Msg) {
			if _, ok := msg.(common.SelectionChangedMsg); ok {
				changed = true
			}
		})
		return changed
	}

	t.Run("QuickSearchMsg", func(t *testing.T) {
		assert.True(t, selectionChanged(model.Update(common.QuickSearchMsg("second"))))
	})

	t.Run("QuickSearchCycle", func(t *testing.T) {
		model.quickSearch = "match"
		assert.True(t, selectionChanged(model.Update(intents.QuickSearchCycle{})))
	})
}

func TestQuickSearch_StreamsUntilMatchFound(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(append([]parser.Row(nil), searchableRows[:2]...), "first")
	model.offScreenRows = append([]parser.Row(nil), model.rows...)
	model.hasMore = true

	model.quickSearch = "third"
	model.applyQuickSearch(0, false)
	assert.NotNil(t, model.pendingSearch, "pending search should be recorded when match is missing and more rows remain")
	assert.Equal(t, 0, model.cursor, "cursor should not move until the match is found")

	test.SimulateModel(model, model.Update(appendRowsBatchMsg{
		rows:    []parser.Row{searchableRows[2]},
		hasMore: false,
		tag:     0,
	}))

	assert.Nil(t, model.pendingSearch, "pending search should clear once the match is found")
	assert.Equal(t, 2, model.cursor, "cursor should move to the streamed match")
}

func TestQuickSearch_StopsWhenStreamExhausted(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(append([]parser.Row(nil), searchableRows[:2]...), "first")
	model.offScreenRows = append([]parser.Row(nil), model.rows...)
	model.hasMore = true

	model.quickSearch = "no-such-query"
	model.applyQuickSearch(0, false)
	assert.NotNil(t, model.pendingSearch)
	initialCursor := model.cursor

	test.SimulateModel(model, model.Update(appendRowsBatchMsg{
		rows:    []parser.Row{searchableRows[2]},
		hasMore: false,
		tag:     0,
	}))

	assert.Nil(t, model.pendingSearch, "pending search should clear when the stream is exhausted")
	assert.Equal(t, initialCursor, model.cursor, "cursor should not move when no match exists")
}

func TestQuickSearch_DoesNotStreamWhenNoMoreRows(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.updateGraphRows(append([]parser.Row(nil), searchableRows[:2]...), "first")
	model.hasMore = false

	model.quickSearch = "no-such-query"
	model.applyQuickSearch(0, false)

	assert.Nil(t, model.pendingSearch, "no pending search should be recorded when the stream has already ended")
}

func TestQuickSearch_ClearCancelsPendingSearch(t *testing.T) {
	model := &Model{
		quickSearch:   "test",
		pendingSearch: &pendingQuickSearch{startIndex: 0},
		baseOp:        operations.NewDefault(),
		rows:          []parser.Row{{Commit: &jj.Commit{ChangeId: "test123"}}},
	}

	_ = model.internalUpdate(intents.RevisionsQuickSearchClear{})

	assert.Equal(t, "", model.quickSearch)
	assert.Nil(t, model.pendingSearch, "clearing the quick search should also drop any pending streamed search")
}

func TestScopes_ExposeQuickSearchScopeWhenSearchActive(t *testing.T) {
	model := &Model{
		quickSearch: "match",
		baseOp:      operations.NewDefault(),
	}

	scopes := model.Scopes()
	assert.NotEmpty(t, scopes)
	assert.Equal(t, bindings.ScopeName("revisions.quick_search"), scopes[0].Name)
}
