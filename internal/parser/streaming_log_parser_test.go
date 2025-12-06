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
	for i := 0; i < 70; i++ {
		lb.Write("*   _PREFIX:abcde_PREFIX:xyrq_PREFIX:false id=abcde author=some@author id=xyrq")
		lb.Write("│   commit " + strconv.Itoa(i))
		lb.Write("~\n")
	}

	reader := strings.NewReader(lb.String())
	controlChannel := make(chan ControlMsg)
	receiver, err := ParseRowsStreaming(reader, controlChannel, 50)

	assert.NoError(t, err)
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
	for i := 0; i < 70; i++ {
		lb.Write("*   _PREFIX:abcde_PREFIX:xyrq_PREFIX:false id=abcde author=some@author id=xyrq")
		lb.Write("│   commit " + strconv.Itoa(i))
		lb.Write("~\n")
	}

	reader := strings.NewReader(lb.String())
	controlChannel := make(chan ControlMsg)
	receiver, err := ParseRowsStreaming(reader, controlChannel, 50)
	assert.NoError(t, err)
	controlChannel <- Close
	_, received := <-receiver
	assert.False(t, received, "expected channel to be closed")
}
