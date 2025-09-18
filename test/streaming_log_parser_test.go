package test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/idursun/jjui/internal/models"
	"github.com/idursun/jjui/internal/parser"
	"github.com/stretchr/testify/assert"
)

func TestParseRowsStreaming_RequestMore(t *testing.T) {
	var lb LogBuilder
	for i := 0; i < 70; i++ {
		lb.Write("*   id=abcde author=some@author id=xyrq")
		lb.Write("│   commit " + strconv.Itoa(i))
		lb.Write("~\n")
	}

	reader := strings.NewReader(lb.String())
	controlChannel := make(chan parser.ControlMsg)
	receiver, err := parser.ParseRowsStreaming(reader, controlChannel, 50)

	assert.NoError(t, err)
	var batch models.RowBatch
	controlChannel <- parser.RequestMore
	batch = <-receiver
	assert.Len(t, batch.Items, 51)
	assert.True(t, batch.HasMore, "expected more rows")

	controlChannel <- parser.RequestMore
	batch = <-receiver
	assert.Len(t, batch.Items, 19)
	assert.False(t, batch.HasMore, "expected no more rows")
}

func TestParseRowsStreaming_Close(t *testing.T) {
	var lb LogBuilder
	for i := 0; i < 70; i++ {
		lb.Write("*   id=abcde author=some@author id=xyrq")
		lb.Write("│   commit " + strconv.Itoa(i))
		lb.Write("~\n")
	}

	reader := strings.NewReader(lb.String())
	controlChannel := make(chan parser.ControlMsg)
	receiver, err := parser.ParseRowsStreaming(reader, controlChannel, 50)
	assert.NoError(t, err)
	controlChannel <- parser.Close
	_, received := <-receiver
	assert.False(t, received, "expected channel to be closed")
}
