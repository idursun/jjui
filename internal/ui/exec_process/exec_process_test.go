package exec_process

import (
	"strings"
	"testing"

	shellwords "github.com/mattn/go-shellwords"
	"github.com/stretchr/testify/assert"
)

// keep splitting behavior on un-quoted input identical to strings.Fields
func TestShellwords_PlainArgs_MatchesFields(t *testing.T) {
	cases := []string{
		"new",
		"new -m description",
		"  git fetch  ",
		"rebase -d main",
	}
	for _, line := range cases {
		t.Run(line, func(t *testing.T) {
			got, err := shellwords.Parse(line)
			assert.NoError(t, err)
			assert.Equal(t, strings.Fields(line), got)
		})
	}
}

// quoted args stay grouped instead of being split on the inner whitespace
func TestShellwords_KeepsQuotedArgs(t *testing.T) {
	got, err := shellwords.Parse("new -m 'add skills'")
	assert.NoError(t, err)
	assert.Equal(t, []string{"new", "-m", "add skills"}, got)

	got, err = shellwords.Parse(`new -m "two words"`)
	assert.NoError(t, err)
	assert.Equal(t, []string{"new", "-m", "two words"}, got)
}

// unbalanced quotes surface as a parse error rather than a mistokenized argv
func TestShellwords_UnbalancedQuoteReturnsError(t *testing.T) {
	_, err := shellwords.Parse("new -m 'unterminated")
	assert.Error(t, err)
}
