package annotation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatAnnotationsMarkdown(t *testing.T) {
	annotations := []Annotation{
		{
			File:     "first.go",
			NewLines: lineRange{Start: 4, End: 5},
			Snippet:  " old\n+new",
			Comment:  "Use a clearer name.",
		},
		{
			File:     "deleted.go",
			OldLines: lineRange{Start: 9, End: 9},
			Snippet:  "-removed",
			Comment:  "Was this deletion intended?",
		},
	}

	assert.Equal(t,
		"### @first.go:4-5\n\n"+
			"```diff\n old\n+new\n```\n\n"+
			"Use a clearer name.\n\n"+
			"### @deleted.go:9\n\n"+
			"```diff\n-removed\n```\n\n"+
			"Was this deletion intended?",
		formatAnnotationsMarkdown(annotations),
	)
}
