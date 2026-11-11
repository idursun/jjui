package annotation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatAnnotationsMarkdown(t *testing.T) {
	annotations := []Annotation{
		{
			ChangeID: "first-change",
			File:     "first.go",
			NewLines: lineRange{Start: 4, End: 5},
			Snippet:  " old\n+new",
			Comment:  "Use a clearer name.",
		},
		{
			ChangeID: "second-change",
			File:     "deleted.go",
			OldLines: lineRange{Start: 9, End: 9},
			Snippet:  "-removed",
			Comment:  "Was this deletion intended?",
		},
	}

	assert.Equal(t,
		"### @first.go (new 4-5)\n\n"+
			"```diff\n old\n+new\n```\n\n"+
			"Use a clearer name.\n\n"+
			"### @deleted.go (old 9)\n\n"+
			"```diff\n-removed\n```\n\n"+
			"Was this deletion intended?",
		formatAnnotationsMarkdown(annotations),
	)
}

func TestFormatAnnotationsMarkdownPreservesBothSides(t *testing.T) {
	annotation := Annotation{
		ChangeID: "mixed-change",
		File:     "example.go",
		OldLines: lineRange{Start: 12, End: 13},
		NewLines: lineRange{Start: 20, End: 21},
		Snippet:  " old\n-new\n+new",
		Comment:  "Please keep this behavior aligned.",
	}

	assert.Equal(t,
		"### @example.go (old 12-13, new 20-21)\n\n"+
			"```diff\n old\n-new\n+new\n```\n\n"+
			"Please keep this behavior aligned.",
		formatAnnotationsMarkdown([]Annotation{annotation}),
	)
}
