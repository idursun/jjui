package parser

import (
	"strconv"
	"strings"
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/screen"
	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestParseRowsStreaming_RequestMore(t *testing.T) {
	var lb test.LogBuilder
	for i := range 70 {
		lb.Write("*   _PREFIX:abcde_PREFIX:xyrq id=abcde author=some@author id=xyrq")
		lb.Write("│   commit " + strconv.Itoa(i))
		lb.Write("~\n")
	}

	reader := strings.NewReader(lb.String())
	controlChannel := make(chan ControlMsg)
	receiver := ParseRowsStreaming(reader, controlChannel, 50, nil)
	var batch RowBatch
	controlChannel <- RequestMore
	batch = <-receiver
	assert.Len(t, batch.Rows, 51)
	assert.True(t, batch.HasMore, "expected more rows")

	controlChannel <- RequestMore
	batch = <-receiver
	assert.Len(t, batch.Rows, 19)
	assert.False(t, batch.HasMore, "expected no more rows")
}

func TestParseRowsStreaming_Close(t *testing.T) {
	var lb test.LogBuilder
	for i := range 70 {
		lb.Write("*   _PREFIX:abcde_PREFIX:xyrq id=abcde author=some@author id=xyrq")
		lb.Write("│   commit " + strconv.Itoa(i))
		lb.Write("~\n")
	}

	reader := strings.NewReader(lb.String())
	controlChannel := make(chan ControlMsg)
	receiver := ParseRowsStreaming(reader, controlChannel, 50, nil)
	controlChannel <- Close
	_, received := <-receiver
	assert.False(t, received, "expected channel to be closed")
}

func TestParseRowPrefixesReadsOptionalMetadataAndLegacyFormat(t *testing.T) {
	for _, tc := range []struct {
		prefix                      string
		bookmarks, local, workspace bool
		root, empty                 bool
	}{
		{"_PREFIX:change_PREFIX:commit_PREFIX:true_PREFIX:false_PREFIX:true_PREFIX:false_PREFIX:true", true, false, true, false, true},
		{"_PREFIX:zzzz_PREFIX:000000_PREFIX:true_PREFIX:true", false, false, false, true, true},
		{"_PREFIX:change_PREFIX:commit_PREFIX:true_PREFIX:false_PREFIX:true", true, false, true, false, false},
		{"_PREFIX:zzzz_PREFIX:000000_PREFIX:true", false, false, false, true, false},
		{"_PREFIX:change_PREFIX:commit", false, false, false, false, false},
	} {
		line := NewGraphRowLine([]*screen.Segment{{Text: "@ " + tc.prefix}})
		index, commit := line.ParseRowPrefixes()
		if index < 0 || commit == nil {
			t.Fatalf("prefix not parsed: %q", tc.prefix)
		}
		if commit.HasBookmarks != tc.bookmarks || commit.HasLocalBookmarks != tc.local || commit.HasWorkspace != tc.workspace || commit.IsRoot() != (tc.root || commit.ChangeId == jj.RootChangeId) || commit.IsEmpty != tc.empty {
			t.Fatalf("metadata = %+v", commit)
		}
	}
}
