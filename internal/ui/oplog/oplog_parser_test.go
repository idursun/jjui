package oplog

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expected []string
	}{
		{
			name:     "single",
			fileName: "single.log",
			expected: []string{"2f7b38830d51"},
		},
		{
			name:     "multi",
			fileName: "multi.log",
			expected: []string{"2f7b38830d51", "ddc6c6080509"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file, err := os.Open(fmt.Sprintf("testdata/%s", test.fileName))
			assert.NoError(t, err)

			rows := parseRows(file)
			var ids []string
			for _, r := range rows {
				ids = append(ids, r.OperationId)
			}
			assert.Equal(t, test.expected, ids)
		})
	}
}

func TestParse_IgnoresIdLikeTextOutsideIdLabel(t *testing.T) {
	log := "<<op_log operation node::○>>  <<op_log operation id short::ddc6c6080509>><<op_log operation::>>\n" +
		"│  <<op_log operation description first_line::abandon>><<op_log operation:: >><<op_log operation description::5ce8658087d8>>\n"
	rows := parseRows(strings.NewReader(log))
	assert.Len(t, rows, 1)
	assert.Equal(t, "ddc6c6080509", rows[0].OperationId)
	assert.Len(t, rows[0].Lines, 2)
}
