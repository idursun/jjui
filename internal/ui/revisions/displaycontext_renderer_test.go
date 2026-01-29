package revisions

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations/describe"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	// Details only renders for the selected commit when Current matches it.
	_ = op.SetSelectedRevision(targetRow.Commit)
	test.SimulateModel(op, op.Init())

	// Render just the target row with the details operation active.
	r := NewDisplayContextRenderer(lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle())
	r.SetSelections(nil)

	width, height := 100, 15
	dl := render.NewDisplayContext()
	viewRect := layout.NewBox(cellbuf.Rect(0, 0, width, height))
	r.Render(dl, []parser.Row{targetRow}, 0, viewRect, op, "", true)

	screen := cellbuf.NewBuffer(width, height)
	dl.Render(screen)
	out := cellbuf.Render(screen)

	// Regression: details list should appear *before* the elided marker line,
	// keeping the marker visually "between" commits rather than above the
	// details list.
	filePos := strings.Index(out, "file.txt")
	elidedPos := strings.Index(out, "elided revisions")
	assert.NotEqual(t, -1, filePos, "expected details list to render file.txt")
	assert.NotEqual(t, -1, elidedPos, "expected fixture to render elided revisions marker")
	assert.Less(t, filePos, elidedPos, "expected details list to render before elided marker")
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

	r := NewDisplayContextRenderer(lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle())
	r.SetSelections(nil)

	width, height := 70, 10
	dl := render.NewDisplayContext()
	viewRect := layout.NewBox(cellbuf.Rect(0, 0, width, height))
	r.Render(dl, []parser.Row{targetRow}, 0, viewRect, op, "", true)

	buf := cellbuf.NewBuffer(width, height)
	dl.Render(buf)
	out := cellbuf.Render(buf)

	// The overlay content should appear in the rendered output.
	assert.Contains(t, out, overlayContent,
		"describe overlay should render for single-line commits")
}
