package test

import (
	"os"
	"strings"
	"testing"

	"github.com/idursun/jjui/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fixtures in testdata are produced by testdata/generate.sh.

func parseFixture(t *testing.T, name string) []parser.Row {
	t.Helper()
	file, err := os.Open("testdata/" + name)
	require.NoError(t, err)
	defer file.Close()
	return parser.ParseRows(file)
}

func ids(rows []parser.Row) []string {
	var result []string
	for _, row := range rows {
		result = append(result, row.Commit.ChangeId+" "+row.Commit.CommitId)
	}
	return result
}

func lineFlags(row parser.Row) []parser.RowLineFlags {
	var flags []parser.RowLineFlags
	for _, line := range row.Lines {
		flags = append(flags, line.Flags)
	}
	return flags
}

const (
	revisionLine    = parser.Revision | parser.Highlightable
	descriptionLine = parser.Description | parser.Highlightable
	contentLine     = parser.Highlightable
)

func TestParser_Graph(t *testing.T) {
	rows := parseFixture(t, "graph.log")
	assert.Equal(t, []string{"yo 4", "v cf", "yq b", "m 7", "zs c6", "q c3", "zz 0"}, ids(rows))
	assert.Equal(t, []int{3, 7, 7, 5, 5, 3, 3}, []int{rows[0].Indent, rows[1].Indent, rows[2].Indent, rows[3].Indent, rows[4].Indent, rows[5].Indent, rows[6].Indent})

	assert.True(t, rows[0].Commit.IsWorkingCopy)
	assert.True(t, rows[0].Commit.IsEmpty)
	assert.False(t, rows[1].Commit.IsWorkingCopy)
	assert.True(t, rows[3].Commit.HasLocalBookmarks, "the long bookmark")
	assert.True(t, rows[5].Commit.HasLocalBookmarks, "main")
	assert.True(t, rows[6].Commit.IsRoot())
	assert.True(t, rows[6].Commit.IsImmutable)
	for _, row := range rows[:6] {
		assert.Equal(t, []parser.RowLineFlags{revisionLine, descriptionLine}, lineFlags(row))
	}
}

func TestParser_ElidedMarkersStayWithTheRevisionAbove(t *testing.T) {
	rows := parseFixture(t, "elided.log")
	assert.Equal(t, []string{"yo 4", "v cf", "q c3"}, ids(rows))
	assert.Equal(t, []parser.RowLineFlags{revisionLine, descriptionLine, parser.Elided, 0}, lineFlags(rows[1]))
	assert.Equal(t, []parser.RowLineFlags{revisionLine, descriptionLine, parser.Elided}, lineFlags(rows[2]), "the end of history marker")
}

func TestParser_OnelineTemplate(t *testing.T) {
	rows := parseFixture(t, "oneline.log")
	assert.Equal(t, []string{"yo 3", "q c3"}, ids(rows))
	assert.Equal(t, []parser.RowLineFlags{revisionLine | parser.Description, contentLine, parser.Elided}, lineFlags(rows[1]))

	var sb strings.Builder
	for _, segment := range rows[1].Extend().Segments {
		sb.WriteString(segment.Text)
	}
	assert.Equal(t, "│  ", sb.String(), "the graph-only line keeps the lane going")
}

func TestParser_DetailedTemplateMarksOnlyDescriptionLines(t *testing.T) {
	rows := parseFixture(t, "detailed.log")
	require.Len(t, rows, 2)
	assert.Equal(t, "yostqsxwqrltovqlrlzszywzslusmuup", rows[0].Commit.ChangeId, "the template renders full ids")
	assert.Equal(t, "339c87bea6a5e02c8af31c8c0aa27e910e813cfe", rows[0].Commit.CommitId)
	assert.Equal(t, []parser.RowLineFlags{
		revisionLine, contentLine, contentLine, contentLine, contentLine,
		descriptionLine, descriptionLine, descriptionLine, descriptionLine,
		contentLine,
	}, lineFlags(rows[0]))

	assert.True(t, rows[1].Commit.HasLocalBookmarks, "local_bookmarks label")
	assert.Equal(t, contentLine, rows[1].Lines[2].Flags, "a ~ in the gutter of a labelled line is not elided")
}

func TestParser_CommitIdOnASeparateLine(t *testing.T) {
	rows := parseFixture(t, "commit-id-second-line.log")
	assert.Equal(t, []string{"yo 3", "q c3"}, ids(rows))
}

func TestParser_Divergent(t *testing.T) {
	rows := parseFixture(t, "divergent.log")
	assert.Equal(t, []string{"n/1 1", "n/0 7", "z 0"}, ids(rows))
}

func TestParser_Evolog(t *testing.T) {
	rows := parseFixture(t, "evolog.log")
	assert.Equal(t, []string{"q 1", "q/1 c3", "q/2 d1", "q/3 c9"}, ids(rows))
	assert.False(t, rows[0].Commit.Hidden)
	assert.True(t, rows[1].Commit.Hidden)
	assert.Equal(t, "c3", rows[1].Commit.GetChangeId(), "hidden commits are addressed by commit id")
}

func TestParser_HiddenRevision(t *testing.T) {
	rows := parseFixture(t, "hidden.log")
	require.Len(t, rows, 1)
	assert.True(t, rows[0].Commit.Hidden)
	assert.Equal(t, "c3", rows[0].Commit.GetChangeId())
}

func TestParser_Disconnected(t *testing.T) {
	var lb LogBuilder
	lb.Revision("", "○", "abcde", "xyrq")
	lb.Description("│  ", "some documentation")
	lb.Graph("~")
	lb.Graph("")
	lb.Revision("", "○", "fghij", "klmn")
	lb.Description("│  ", "another commit")
	lb.Graph("~")
	rows := parser.ParseRows(strings.NewReader(lb.String()))
	assert.Equal(t, []string{"abcde xyrq", "fghij klmn"}, ids(rows))
}

func TestParser_Extend(t *testing.T) {
	var lb LogBuilder
	lb.Revision("", "○", "abcde", "xyrq")
	lb.Description("│  ", "some documentation")
	rows := parser.ParseRows(strings.NewReader(lb.String()))
	require.Len(t, rows, 1)
	assert.Len(t, rows[0].Extend().Segments, 3)
}

func TestParser_WorkingCopy(t *testing.T) {
	var lb LogBuilder
	lb.Revision("", "○", "abcde", "xyrq")
	lb.Description("│  ", "some documentation")
	lb.Revision("", "@", "kdys", "12cd")
	lb.Description("│  ", "some documentation")
	rows := parser.ParseRows(strings.NewReader(lb.String()))
	require.Len(t, rows, 2)
	assert.False(t, rows[0].Commit.IsWorkingCopy)
	assert.True(t, rows[1].Commit.IsWorkingCopy)
}
