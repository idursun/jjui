package test

import (
	"os"
	"strings"
	"testing"

	"github.com/idursun/jjui/internal/parser"
	"github.com/stretchr/testify/assert"
)

// In order to get a log output for testing, run the following command:
// jj log -T "stringify('_PREFIX:' ++ separate('_PREFIX:', change_id.shortest(), commit_id.shortest(), divergent)) ++ ' ' ++ builtin_log_compact" --color always > test/testdata/your_test_case.log

func TestParser_Parse(t *testing.T) {
	file, _ := os.Open("testdata/output.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 11)
}

func TestParser_Parse_WorkingCopyCommit(t *testing.T) {
	file, _ := os.Open("testdata/working-copy-commit.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 9)
	row := rows[5]
	assert.True(t, row.Commit.IsWorkingCopy)
}

func TestParser_Parse_DivergentLog(t *testing.T) {
	file, _ := os.Open("testdata/divergent.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 3)
	assert.True(t, strings.HasSuffix(rows[0].Commit.ChangeId, "/0"))
	assert.True(t, strings.HasSuffix(rows[1].Commit.ChangeId, "/1"))
	assert.Equal(t, "zo/2", rows[2].Commit.ChangeId)
}

// TODO: what does no-commit-id mean?
// `testdata/no-commit-id.log` has commit id
func TestParser_Parse_NoCommitId(t *testing.T) {
	file, _ := os.Open("testdata/no-commit-id.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 1)
}

func TestParser_Parse_ShortId(t *testing.T) {
	file, _ := os.Open("testdata/short-id.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 2)
	assert.Equal(t, "X", rows[0].Commit.ChangeId)
	assert.Equal(t, "E", rows[0].Commit.CommitId)
	assert.Equal(t, "T", rows[1].Commit.ChangeId)
	assert.Equal(t, "79", rows[1].Commit.CommitId)
}

func TestParser_Parse_SingleLineWithDescription(t *testing.T) {
	file, _ := os.Open("testdata/single-line-with-description.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 1)
	assert.Equal(t, "x", rows[0].Commit.ChangeId)
	assert.Equal(t, "4", rows[0].Commit.CommitId)
}

func TestParser_Parse_CommitIdOnASeparateLine(t *testing.T) {
	file, _ := os.Open("testdata/commit-id.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 1)
	assert.Equal(t, "o", rows[0].Commit.ChangeId)
	assert.Equal(t, "5", rows[0].Commit.CommitId)
}

func TestParser_Parse_ConflictedLongIds(t *testing.T) {
	file, _ := os.Open("testdata/conflicted-change-id.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 3)
	assert.Equal(t, "p/0", rows[0].Commit.ChangeId)
	assert.Equal(t, "q/0", rows[1].Commit.ChangeId)
	assert.Equal(t, "t/0", rows[2].Commit.ChangeId)
}

func TestParser_Parse_Disconnected(t *testing.T) {
	var lb LogBuilder
	lb.Write("*   _PREFIX:abcde_PREFIX:xyrq_PREFIX:false id=abcde author=some@author id=xyrq")
	lb.Write("│   some documentation")
	lb.Write("~\n")
	lb.Write("*   _PREFIX:abcde_PREFIX:xyrq_PREFIX:false id=abcde author=some@author id=xyrq")
	lb.Write("│   another commit")
	lb.Write("~\n")
	rows := parser.ParseRows(strings.NewReader(lb.String()))
	assert.Len(t, rows, 2)
	assert.Equal(t, "abcde", rows[0].Commit.ChangeId)
}

func TestParser_Parse_Extend(t *testing.T) {
	var lb LogBuilder
	lb.Write("*   _PREFIX:abcde_PREFIX:xyrq_PREFIX:false id=abcde author=some@author id=xyrq")
	lb.Write("│   some documentation")

	rows := parser.ParseRows(strings.NewReader(lb.String()))
	assert.Len(t, rows, 1)
	row := rows[0]

	extended := row.Extend()
	assert.Len(t, extended.Segments, 2)
}

func TestParser_Parse_WorkingCopy_1(t *testing.T) {
	var lb LogBuilder
	lb.Write("*   _PREFIX:abcde_PREFIX:xyrq_PREFIX:false id=abcde author=some@author id=xyrq")
	lb.Write("│   some documentation")
	lb.Write("@   _PREFIX:kdys_PREFIX:12cd_PREFIX:false short_id=kdys author=some@author id=12cd")
	lb.Write("│   some documentation")

	rows := parser.ParseRows(strings.NewReader(lb.String()))
	assert.Len(t, rows, 2)
	row := rows[1]

	assert.True(t, row.Commit.IsWorkingCopy)
}

func TestParser_ChangeIdLikeDescription(t *testing.T) {
	file, _ := os.Open("testdata/change-id-like-description.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 1)
}

func TestParser_WorkingCopyOnBranch(t *testing.T) {
	file, _ := os.Open("testdata/working-copy-on-branch.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 10)
	assert.Equal(t, "tr", rows[0].Commit.ChangeId)
	assert.Equal(t, "83", rows[0].Commit.CommitId)
	assert.Equal(t, "no", rows[1].Commit.ChangeId)
	assert.Equal(t, "11", rows[1].Commit.CommitId)
	assert.Equal(t, "yl", rows[2].Commit.ChangeId)
	assert.Equal(t, "d", rows[2].Commit.CommitId)
	assert.Equal(t, "kl", rows[3].Commit.ChangeId)
	assert.Equal(t, "6", rows[3].Commit.CommitId)
}

func TestParser_VeryLongBookmark(t *testing.T) {
	file, _ := os.Open("testdata/long-bookmark.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 5)
	assert.Equal(t, "nr", rows[0].Commit.ChangeId)
	assert.Equal(t, "8", rows[0].Commit.CommitId)
	assert.Equal(t, "lu", rows[1].Commit.ChangeId)
	assert.Equal(t, "fd", rows[1].Commit.CommitId)
	assert.Equal(t, "wto", rows[3].Commit.ChangeId)
	assert.Equal(t, "5", rows[3].Commit.CommitId)
}

func TestParser_DivergentChangeID(t *testing.T) {
	file, _ := os.Open("testdata/divergent-change-id.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 2)
	assert.Equal(t, "omv/0", rows[0].Commit.ChangeId)
	assert.Equal(t, "f99", rows[0].Commit.CommitId)
	assert.Equal(t, "omv/1", rows[1].Commit.ChangeId)
	assert.Equal(t, "43bd", rows[1].Commit.CommitId)
}

func TestParser_DivergentChangeIDShort(t *testing.T) {
	file, _ := os.Open("testdata/divergent-short-change-id.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 2)
	assert.Equal(t, "omv/0", rows[0].Commit.ChangeId)
	assert.Equal(t, "f99", rows[0].Commit.CommitId)
	assert.Equal(t, "omv/1", rows[1].Commit.ChangeId)
	assert.Equal(t, "43bd", rows[1].Commit.CommitId)
}

func TestParser_ChangeIDCommitIDSameColor(t *testing.T) {
	file, _ := os.Open("testdata/change-commit-same-color.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 3)
	assert.Equal(t, "vvr", rows[0].Commit.ChangeId)
	assert.Equal(t, "ae", rows[0].Commit.CommitId)
	assert.Equal(t, "l", rows[1].Commit.ChangeId)
	assert.Equal(t, "6e", rows[1].Commit.CommitId)
	assert.Equal(t, "xv", rows[2].Commit.ChangeId)
	assert.Equal(t, "fa", rows[2].Commit.CommitId)
}

func TestParser_Evolog(t *testing.T) {
	file, _ := os.Open("testdata/evolog.log")
	rows := parser.ParseRows(file)
	assert.Len(t, rows, 2)
	assert.Equal(t, "l", rows[0].Commit.ChangeId)
	assert.Equal(t, "98", rows[0].Commit.CommitId)
}
