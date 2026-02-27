package revisions

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/render"
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

// mockNonFocusableOperation is a mock operation that is never focused, editing, or overlay
type mockNonFocusableOperation struct{}

func (m *mockNonFocusableOperation) Render(commit *jj.Commit, renderPosition operations.RenderPosition) string {
	return ""
}

func (m *mockNonFocusableOperation) RenderToDisplayContext(_ *render.DisplayContext, _ *jj.Commit, _ operations.RenderPosition, _ cellbuf.Rectangle, _ cellbuf.Position) int {
	return 0
}

func (m *mockNonFocusableOperation) DesiredHeight(_ *jj.Commit, _ operations.RenderPosition) int {
	return 0
}

func (m *mockNonFocusableOperation) Name() string {
	return "mock"
}

func (m *mockNonFocusableOperation) Init() tea.Cmd {
	return nil
}

func (m *mockNonFocusableOperation) Update(msg tea.Msg) tea.Cmd {
	return nil
}

func (m *mockNonFocusableOperation) ViewRect(_ *render.DisplayContext, _ layout.Box) {}

func (m *mockNonFocusableOperation) IsFocused() bool {
	return false
}

func (m *mockNonFocusableOperation) IsEditing() bool {
	return false
}

func (m *mockNonFocusableOperation) IsOverlay() bool {
	return false
}

// TestQuickSearch_ClearIntentClearsSearch tests that clear intent clears the search.
func TestQuickSearch_ClearIntentClearsSearch(t *testing.T) {
	model := &Model{
		quickSearch: "test",
		op:          &mockNonFocusableOperation{},
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
