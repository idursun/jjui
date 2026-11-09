package revisions

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

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
	op := details.NewOperation(ctx, targetRow.Commit, details.FileChecks{
		IsChecked: func(string, jj.FileName) bool { return false },
		Toggle:    func(string, jj.FileName) {},
	})
	test.SimulateModel(op, op.Init())

	// Render just the target row with the details operation active.
	r := NewDisplayContextRenderer()

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
	r.selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("4"))

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

	// The overlay is appended after the single parsed revision line, so it has
	// no source description line for a separate highlight pass to discover.
	// Its rendered rectangle must be highlighted directly.
	overlayCell := buf.CellAt(0, 1)
	require.NotNil(t, overlayCell)
	assert.NotNil(t, overlayCell.Style.Bg,
		"describe overlay should inherit the selected revision background")
}

func TestDisplayContextRenderer_SetBookmarkRegistersInlineCursor(t *testing.T) {
	row := parser.Row{
		Commit: &jj.Commit{ChangeId: "abc123", CommitId: "def456"},
		Lines: []*parser.GraphRowLine{
			{
				Gutter:   parser.GraphGutter{Segments: []*screen.Segment{{Text: "|"}}},
				Segments: []*screen.Segment{{Text: "def456", Labels: []string{"commit_id"}}, {Text: " bookmark target"}},
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

func TestDisplayContextRenderer_SetBookmarkInputGoesBeforeLabelledCommitId(t *testing.T) {
	// The shortest commit id "2" is also a prefix of the timestamp.
	log := "<<log commit node mutable::○>>  <<log commit mutable change_id shortest prefix::m>><<log commit mutable change_id shortest rest::mypmnqu>><<log commit mutable:: >>" +
		"<<log commit mutable committer timestamp local format::2025-05-22 21:59:38>><<log commit mutable:: >>" +
		"<<log commit mutable commit_id shortest prefix::2>><<log commit mutable commit_id shortest rest::8c8ae4a>><<log commit mutable::>>\n"
	rows := parser.ParseRows(strings.NewReader(log))
	require.Len(t, rows, 1)

	op := bookmark.NewSetBookmarkOperation(test.NewTestContext(test.NewTestCommandRunner(t)), rows[0].Commit.GetChangeId(), "NAME")
	r := NewDisplayContextRenderer()
	dl := render.NewDisplayContext()
	r.Render(dl, rows, 0, layout.NewBox(layout.Rect(0, 0, 80, 3)), op, nil, false, "", true)
	screen := uv.NewScreenBuffer(80, 3)
	dl.Render(screen)

	firstLine := strings.Split(screen.Render(), "\n")[0]
	assert.Contains(t, firstLine, "2025-05-22 21:59:38 NAME  28c8ae4a")
}

func TestDisplayContextRenderer_DescribeOverlaysOnlyDescriptionLines(t *testing.T) {
	f, err := os.Open("../../../test/testdata/detailed.log")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	rows := parser.ParseRows(f)
	require.NotEmpty(t, rows)
	targetRow := rows[0]

	const overlayContent = "[describe overlay content]"
	commandRunner := test.NewTestCommandRunner(t)
	commandRunner.Expect(jj.GetDescription(targetRow.Commit.GetChangeId())).SetOutput([]byte(overlayContent))
	defer commandRunner.Verify()
	op := describe.NewOperation(test.NewTestContext(commandRunner), targetRow.Commit)

	r := NewDisplayContextRenderer()
	width, height := 90, 20
	dl := render.NewDisplayContext()
	r.Render(dl, []parser.Row{targetRow}, 0, layout.NewBox(layout.Rect(0, 0, width, height)), op, nil, false, "", true)
	buf := uv.NewScreenBuffer(width, height)
	dl.Render(buf)
	out := buf.Render()

	// The header lines of the detailed template stay; the description goes.
	for _, kept := range []string{"Change ID:", "Author   :", "Committer:"} {
		assert.Contains(t, out, kept)
	}
	assert.NotContains(t, out, "with a longer body")
	assert.Less(t, strings.Index(out, "Committer:"), strings.Index(out, overlayContent))
}

func TestDisplayContextRenderer_RenameBookmarkEditsTheNameInPlace(t *testing.T) {
	log := "<<log commit node mutable::○>>  <<log commit mutable change_id shortest prefix::m>><<log commit mutable:: >>" +
		"<<log commit mutable bookmarks name::second>><<log commit mutable bookmarks:: >>" +
		"<<log commit mutable bookmarks name::second>><<log commit mutable bookmarks::@>><<log commit mutable bookmarks remote::origin>><<log commit mutable:: >>" +
		"<<log commit mutable commit_id shortest prefix::28>><<log commit mutable::>>\n"
	rows := parser.ParseRows(strings.NewReader(log))
	require.Len(t, rows, 1)

	op := bookmark.NewRenameBookmarkOperation(test.NewTestContext(test.NewTestCommandRunner(t)), rows[0].Commit.GetChangeId(), "second")
	test.SimulateModel(op, test.Type("-x"))

	r := NewDisplayContextRenderer()
	dl := render.NewDisplayContext()
	r.Render(dl, rows, 0, layout.NewBox(layout.Rect(0, 0, 80, 3)), op, op, false, "", true)
	screen := uv.NewScreenBuffer(80, 3)
	dl.Render(screen)
	firstLine := ansi.Strip(strings.Split(screen.Render(), "\n")[0])

	// The input keeps a cell after the text for the cursor.
	assert.Equal(t, "○  m second-x  second@origin 28", strings.TrimRight(firstLine, " "))
	cursor := dl.Cursor()
	require.NotNil(t, cursor)
	nameColumn := ansi.StringWidth(firstLine[:strings.Index(firstLine, "second-x")])
	assert.Equal(t, nameColumn+len("second-x"), cursor.Position.X)
}
