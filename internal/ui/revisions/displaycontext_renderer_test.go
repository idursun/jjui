package revisions

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations"
	"github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/describe"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type beforeMarkerOperation struct {
	changeID string
	content  string
}

func (o beforeMarkerOperation) Init() tea.Cmd { return nil }

func (o beforeMarkerOperation) Update(tea.Msg) tea.Cmd { return nil }

func (o beforeMarkerOperation) ViewRect(*render.DisplayContext, layout.Box) {}

func (o beforeMarkerOperation) Render(commit *jj.Commit, pos operations.RenderPosition) string {
	if pos == operations.RenderPositionBefore && commit.GetChangeId() == o.changeID {
		return o.content
	}
	return ""
}

func (o beforeMarkerOperation) Name() string { return "before.marker" }

func TestDisplayContextRenderer_DetailsRendersBeforeElidedMarker(t *testing.T) {
	f, err := os.Open("testdata/jj-log-with-elided.log")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	rows := parser.ParseRows(f)
	require.NotEmpty(t, rows)

	// rows[1] in `jj-log-with-elided.log` has (~ elided revision) below
	targetRow := rows[1]
	require.NotNil(t, targetRow.Commit)

	// Prepare details operation with a file list.
	const statusOutput = "false $\nM file.txt\n"
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.Snapshot())
	commandRunner.Expect(jj.Status(targetRow.Commit.GetChangeId())).SetOutput([]byte(statusOutput))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	op := details.NewOperation(ctx, targetRow.Commit)
	test.SimulateModel(op, op.Init())

	// Render just the target row with the details operation active.
	r := NewDisplayContextRenderer()
	r.SetSelections(nil)

	width, height := 100, 15
	dl := render.NewDisplayContext()
	viewRect := layout.NewBox(layout.Rect(0, 0, width, height))
	r.Render(dl, []parser.Row{targetRow}, 0, viewRect, op, nil, false, "", true)

	screen := uv.NewScreenBuffer(width, height)
	dl.Render(screen)
	out := screen.Render()

	// Regression: details list should appear *before* the elided marker line,
	// keeping the marker visually "between" commits rather than above the
	// details list.
	filePos := strings.Index(out, "file.txt")
	elidedPos := strings.Index(out, "elided revisions")
	assert.NotEqual(t, -1, filePos, "expected details list to render file.txt")
	assert.NotEqual(t, -1, elidedPos, "expected fixture to render elided revisions marker")
	assert.Less(t, filePos, elidedPos, "expected details list to render before elided marker")
}

func TestDisplayContextRenderer_RendersBeforePositionForNonSelectedRows(t *testing.T) {
	rows := []parser.Row{
		{
			Commit: &jj.Commit{ChangeId: "source", CommitId: "111111"},
			Lines: []*parser.GraphRowLine{
				{
					Gutter:   parser.GraphGutter{Segments: []*screen.Segment{{Text: "@"}}},
					Segments: []*screen.Segment{{Text: "111111 source commit"}},
					Flags:    parser.Revision,
				},
			},
		},
		{
			Commit: &jj.Commit{ChangeId: "target", CommitId: "222222"},
			Lines: []*parser.GraphRowLine{
				{
					Gutter:   parser.GraphGutter{Segments: []*screen.Segment{{Text: "○"}}},
					Segments: []*screen.Segment{{Text: "222222 target commit"}},
					Flags:    parser.Revision | parser.Highlightable,
				},
			},
		},
	}
	rows[1].Previous = &rows[0]

	r := NewDisplayContextRenderer()
	dl := render.NewDisplayContext()
	viewRect := layout.NewBox(layout.Rect(0, 0, 80, 5))
	op := beforeMarkerOperation{changeID: "source", content: "<< from >>"}

	r.Render(dl, rows, 1, viewRect, op, nil, false, "", true)

	buf := uv.NewScreenBuffer(80, 5)
	dl.Render(buf)
	out := buf.Render()

	markerPos := strings.Index(out, "<< from >>")
	sourcePos := strings.Index(out, "source commit")
	targetPos := strings.Index(out, "target commit")
	require.NotEqual(t, -1, markerPos, "expected before marker to render for non-selected row")
	require.NotEqual(t, -1, sourcePos, "expected source row to remain visible")
	require.NotEqual(t, -1, targetPos, "expected target row to render after measured source height")
	assert.Less(t, markerPos, sourcePos)
	assert.Less(t, sourcePos, targetPos)
}

// Tests that the description overlay renders correctly even when the commit has only a single line.
// This is the case with log_oneline templates, which render like:
//
//	○  ntqpysmy user@example.com (3 days ago) abc123 fix: delete all code
//
// Regression test for: https://github.com/idursun/jjui/issues/280
func TestDisplayContextRenderer_SingleRowDescriptionOverlay(t *testing.T) {
	f, err := os.Open("testdata/single-line-log.log")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	rows := parser.ParseRows(f)
	require.NotEmpty(t, rows)

	// This file contains a single-line commit (no description line below).
	targetRow := rows[0]
	require.NotNil(t, targetRow.Commit)
	require.Len(t, targetRow.Lines, 1, "expected single-line commit for this test")

	// Create describe operation with distinctive content that should appear in overlay.
	const overlayContent = "[describe overlay content]"
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription(targetRow.Commit.GetChangeId())).SetOutput([]byte(overlayContent))
	defer commandRunner.Verify()

	ctx := test.NewTestContext(commandRunner)
	op := describe.NewOperation(ctx, targetRow.Commit)

	r := NewDisplayContextRenderer()
	r.SetSelections(nil)

	width, height := 70, 10
	dl := render.NewDisplayContext()
	viewRect := layout.NewBox(layout.Rect(0, 0, width, height))
	r.Render(dl, []parser.Row{targetRow}, 0, viewRect, op, nil, false, "", true)

	buf := uv.NewScreenBuffer(width, height)
	dl.Render(buf)
	out := buf.Render()

	// The overlay content should appear in the rendered output.
	assert.Contains(t, out, overlayContent,
		"describe overlay should render for single-line commits")
}

func TestDisplayContextRenderer_SetBookmarkRegistersInlineCursor(t *testing.T) {
	row := parser.Row{
		Commit: &jj.Commit{ChangeId: "abc123", CommitId: "def456"},
		Lines: []*parser.GraphRowLine{
			{
				Gutter:   parser.GraphGutter{Segments: []*screen.Segment{{Text: "|"}}},
				Segments: []*screen.Segment{{Text: "def456 bookmark target"}},
				Flags:    parser.Revision,
			},
		},
	}

	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	op := bookmark.NewSetBookmarkOperation(ctx, "abc123", "main")

	r := NewDisplayContextRenderer()
	dl := render.NewDisplayContext()
	viewRect := layout.NewBox(layout.Rect(5, 3, 60, 5))
	r.Render(dl, []parser.Row{row}, 0, viewRect, op, nil, false, "", true)

	cursor := dl.Cursor()
	require.NotNil(t, cursor)

	inlineCursor := op.InlineCursor(row.Commit, operations.RenderBeforeCommitId)
	require.NotNil(t, inlineCursor)
	assert.Equal(t, viewRect.R.Min.X+1+inlineCursor.Position.X, cursor.Position.X)
	assert.Equal(t, viewRect.R.Min.Y+inlineCursor.Position.Y, cursor.Position.Y)
}
