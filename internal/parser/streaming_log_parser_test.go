package parser

import (
	"strconv"
	"strings"
	"testing"

	"github.com/idursun/jjui/test"
	"github.com/stretchr/testify/assert"
)

func TestParseRowsStreaming_RequestMore(t *testing.T) {
	var lb test.LogBuilder
	for i := range 70 {
		lb.Revision("", "○", "abcde", "xyrq")
		lb.Description("│  ", "commit "+strconv.Itoa(i))
		lb.Graph("~")
		lb.Graph("")
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
		lb.Revision("", "○", "abcde", "xyrq")
		lb.Description("│  ", "commit "+strconv.Itoa(i))
		lb.Graph("~")
		lb.Graph("")
	}

	reader := strings.NewReader(lb.String())
	controlChannel := make(chan ControlMsg)
	receiver := ParseRowsStreaming(reader, controlChannel, 50, nil)
	controlChannel <- Close
	_, received := <-receiver
	assert.False(t, received, "expected channel to be closed")
}
