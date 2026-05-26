package revisions

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextObjectsForRowExtractsDefaultCompactSections(t *testing.T) {
	row := textObjectTestRow()

	objects := textObjectsForRow(row)

	require.Len(t, objects, 6)
	assert.Equal(t, textObjectGraph, objects[0].Kind)
	assert.Equal(t, textObjectAuthor, objects[1].Kind)
	assert.Equal(t, "user@example.com", objects[1].Value)
	assert.Equal(t, textObjectDate, objects[2].Kind)
	assert.Equal(t, "3 weeks ago", objects[2].Value)
	assert.Equal(t, textObjectBookmark, objects[3].Kind)
	assert.Equal(t, "main", objects[3].Value)
	assert.Equal(t, textObjectBookmark, objects[4].Kind)
	assert.Equal(t, "feature", objects[4].Value)
	assert.Equal(t, textObjectDescription, objects[5].Kind)
	assert.Equal(t, "subject line", objects[5].Value)
}

func TestTextObjectsForRowIncludesPlaceholderBookmark(t *testing.T) {
	row := textObjectTestRow()
	row.Lines[0].Segments = []*screen.Segment{
		{Text: "abc", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))},
		{Text: " rest", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("8"))},
		{Text: " user@example.com ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("3"))},
		{Text: "3 weeks ago ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("6"))},
		{Text: "def", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))},
	}

	objects := textObjectsForRow(row)

	require.Len(t, objects, 5)
	assert.Equal(t, textObjectBookmark, objects[3].Kind)
	assert.Equal(t, "", objects[3].Value)
	assert.Equal(t, 0, objects[3].Index)
	assert.Equal(t, 1, objects[3].width)
	assert.Equal(t, textObjectDescription, objects[4].Kind)
}

func TestTextObjectsForCurrentRowRecognizesBrightMetadata(t *testing.T) {
	row := textObjectTestRow()
	row.Lines[0].Segments = []*screen.Segment{
		{Text: "abc", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))},
		{Text: " rest", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("8"))},
		{Text: " user@example.com ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("3"))},
		{Text: "3 weeks ago ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("14"))},
		{Text: "current-bookmark ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("13"))},
		{Text: "def", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))},
	}

	objects := textObjectsForRow(row)

	require.Len(t, objects, 5)
	assert.Equal(t, textObjectBookmark, objects[3].Kind)
	assert.Equal(t, "current-bookmark", objects[3].Value)
	assert.Greater(t, objects[3].x, objects[2].x)
	assert.Equal(t, textObjectDescription, objects[4].Kind)
	assert.Equal(t, "subject line", objects[4].Value)
}

func TestTextObjectsForSingleLineDescriptionAfterCommitID(t *testing.T) {
	row := textObjectTestRow()
	row.Lines = []*parser.GraphRowLine{
		{
			Gutter: parser.GraphGutter{Segments: []*screen.Segment{
				{Text: "○"},
				{Text: " "},
			}},
			Segments: []*screen.Segment{
				{Text: "abc", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("13"))},
				{Text: " rest", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("8"))},
				{Text: " user@example.com ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("3"))},
				{Text: "3 weeks ago ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("14"))},
				{Text: "def", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))},
				{Text: " subject line"},
			},
			Flags: parser.Revision | parser.Highlightable,
		},
	}

	objects := textObjectsForRow(row)

	require.Len(t, objects, 5)
	assert.Equal(t, textObjectBookmark, objects[3].Kind)
	assert.Equal(t, "", objects[3].Value)
	assert.Equal(t, textObjectDescription, objects[4].Kind)
	assert.Equal(t, "subject line", objects[4].Value)
	assert.Greater(t, objects[4].x, objects[3].x)
}

func TestTextObjectNavigationUpdatesFocusedObject(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.rows = []parser.Row{textObjectTestRow()}

	model.updateFocusedObject()
	require.NotNil(t, ctx.FocusedObject)
	assert.Equal(t, textObjectGraph, ctx.FocusedObject.Kind)

	model.navigateFocusedObject(1)
	require.NotNil(t, ctx.FocusedObject)
	assert.Equal(t, textObjectAuthor, ctx.FocusedObject.Kind)

	model.navigateFocusedObject(3)
	require.NotNil(t, ctx.FocusedObject)
	assert.Equal(t, textObjectBookmark, ctx.FocusedObject.Kind)
	assert.Equal(t, "feature", ctx.FocusedObject.Value)
	assert.Equal(t, 1, ctx.FocusedObject.Index)
}

func TestFocusGraphObjectMovesBackToGlyph(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	model.rows = []parser.Row{textObjectTestRow()}
	model.focusedObjectKind = textObjectDescription
	model.updateFocusedObject()
	require.NotNil(t, ctx.FocusedObject)
	require.Equal(t, textObjectDescription, ctx.FocusedObject.Kind)

	model.focusGraphObject()

	require.NotNil(t, ctx.FocusedObject)
	assert.Equal(t, textObjectGraph, ctx.FocusedObject.Kind)
	assert.Equal(t, bindings.ScopeName(""), model.focusedObjectScope())
}

func TestTextObjectNavigationCanFocusPlaceholderBookmark(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	row := textObjectTestRow()
	row.Lines[0].Segments = []*screen.Segment{
		{Text: "abc", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))},
		{Text: " user@example.com ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("3"))},
		{Text: "3 weeks ago ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("6"))},
		{Text: "def", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))},
	}
	model.rows = []parser.Row{row}
	model.updateFocusedObject()

	model.navigateFocusedObject(3)

	require.NotNil(t, ctx.FocusedObject)
	assert.Equal(t, textObjectBookmark, ctx.FocusedObject.Kind)
	assert.Equal(t, "", ctx.FocusedObject.Value)
	assert.Equal(t, bindings.ScopeName("revisions.bookmark"), model.focusedObjectScope())
}

func TestNavigatePreservesFocusedTextObjectAcrossRows(t *testing.T) {
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	model := New(ctx)
	row1 := textObjectTestRow()
	row2 := textObjectTestRow()
	row2.Commit = &jj.Commit{ChangeId: "xyz", CommitId: "uvw"}
	row2.Lines[0].Segments[0].Text = "xyz"
	row2.Lines[0].Segments[len(row2.Lines[0].Segments)-1].Text = "uvw"
	model.rows = []parser.Row{row1, row2}
	model.focusedObjectKind = textObjectBookmark
	model.focusedObjectIndex = 1
	model.updateFocusedObject()
	require.NotNil(t, ctx.FocusedObject)
	require.Equal(t, "feature", ctx.FocusedObject.Value)

	model.Update(intents.Navigate{Delta: 1})

	assert.Equal(t, 1, model.cursor)
	require.NotNil(t, ctx.FocusedObject)
	assert.Equal(t, textObjectBookmark, ctx.FocusedObject.Kind)
	assert.Equal(t, "feature", ctx.FocusedObject.Value)
	assert.Equal(t, 1, ctx.FocusedObject.Index)
	assert.Equal(t, "xyz", ctx.FocusedObject.ChangeId)
	assert.Equal(t, "uvw", ctx.FocusedObject.CommitId)
}

func TestDisplayContextRendererPlacesCursorOnFocusedObject(t *testing.T) {
	row := textObjectTestRow()
	objects := textObjectsForRow(row)
	require.GreaterOrEqual(t, len(objects), 2)
	focused := objects[1]

	r := NewDisplayContextRenderer()
	dl := render.NewDisplayContext()
	viewRect := layout.NewBox(layout.Rect(5, 3, 80, 8))
	r.Render(dl, []parser.Row{row}, 0, viewRect, nil, nil, false, "", true, &focused)

	cursor := dl.Cursor()
	require.NotNil(t, cursor)
	assert.Equal(t, viewRect.R.Min.X+focused.x, cursor.Position.X)
	assert.Equal(t, viewRect.R.Min.Y+focused.line, cursor.Position.Y)
}

func textObjectTestRow() parser.Row {
	return parser.Row{
		Commit: &jj.Commit{ChangeId: "abc", CommitId: "def"},
		Lines: []*parser.GraphRowLine{
			{
				Gutter: parser.GraphGutter{Segments: []*screen.Segment{
					{Text: "○"},
					{Text: " "},
				}},
				Segments: []*screen.Segment{
					{Text: "abc", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))},
					{Text: " rest", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("8"))},
					{Text: " user@example.com ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("3"))},
					{Text: "3 weeks ago ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("6"))},
					{Text: "main feature ", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("5"))},
					{Text: "def", Style: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))},
				},
				Flags: parser.Revision | parser.Highlightable,
			},
			{
				Gutter: parser.GraphGutter{Segments: []*screen.Segment{
					{Text: "│"},
					{Text: " "},
				}},
				Segments: []*screen.Segment{{Text: "subject line"}},
				Flags:    parser.Highlightable,
			},
		},
	}
}
