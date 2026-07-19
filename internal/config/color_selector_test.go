package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseColorSelector_NormalizesSelectedSyntax(t *testing.T) {
	tests := map[string]string{
		"selected":                          ":selected",
		":selected":                         ":selected",
		"revisions selected":                "revisions:selected",
		"revisions:selected":                "revisions:selected",
		"revisions selected text":           "revisions text:selected",
		"revisions text:selected":           "revisions text:selected",
		"revset completion selected dimmed": "revset completion dimmed:selected",
		"revset completion dimmed:selected": "revset completion dimmed:selected",
		"revisions text":                    "revisions text",
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			assert.Equal(t, expected, ParseColorSelector(input).Key())
		})
	}
}

func TestNormalizeColorSelectors_SuffixSyntaxWinsWithinLayer(t *testing.T) {
	colors := NormalizeColorSelectors(map[string]Color{
		"menu selected text": {Fg: "red"},
		"menu text:selected": {Fg: "blue"},
	})

	assert.Equal(t, map[string]Color{
		"menu text:selected": {Fg: "blue"},
	}, colors)
}

func TestParseColorSelector_LeavesUnknownStateSuffixUntouched(t *testing.T) {
	selector := ParseColorSelector("revisions text:focused")

	assert.False(t, selector.HasVariant(SelectedVariant))
	assert.Equal(t, "revisions text:focused", selector.Key())
}
