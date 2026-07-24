package annotation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRevisionTargets(t *testing.T) {
	assert.Equal(t, []revisionTarget{
		{ChangeID: "first", Description: "First description"},
		{ChangeID: "second", Description: "(no description set)"},
		{ChangeID: "third", Description: "Description with\ttab"},
	}, parseRevisionTargets(
		"first\tFirst description\n"+
			"second\t\n"+
			"\n"+
			"third\tDescription with\ttab\r\n",
	))
}
