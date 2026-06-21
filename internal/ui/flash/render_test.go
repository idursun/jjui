package flash

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestCardRenderer_InterpretsCarriageReturnsInCommandError(t *testing.T) {
	rendered := NewCardRenderer().RenderMessage(
		"jj git push",
		"",
		errors.New("git: first\rgit: second\r\nfatal: failed"),
		80,
	)
	plain := ansi.Strip(rendered)

	assert.NotContains(t, plain, "\r")
	assert.NotContains(t, plain, "git: first")
	assert.Contains(t, plain, "git: second")
	assert.Contains(t, plain, "fatal: failed")

	for _, line := range strings.Split(plain, "\n") {
		assert.True(t, strings.HasPrefix(line, "│") || strings.HasPrefix(line, "┌") || strings.HasPrefix(line, "└"))
		assert.True(t, strings.HasSuffix(line, "│") || strings.HasSuffix(line, "┐") || strings.HasSuffix(line, "┘"))
	}
}

func TestCardRenderer_InterpretsCarriageReturnsInHistoryOutput(t *testing.T) {
	rendered := NewCardRenderer().RenderHistoryEntry(commandHistoryEntry{
		Command: "jj git push",
		Text:    "first\rsecond\r\nthird",
	}, 80, true)
	plain := ansi.Strip(rendered)

	assert.NotContains(t, plain, "\r")
	assert.NotContains(t, plain, "first")
	assert.Regexp(t, `(?m)^│ second +│$`, plain)
	assert.Regexp(t, `(?m)^│ third +│$`, plain)
}

func TestCardRenderer_RendersGitSidebandOutputLikeTerminal(t *testing.T) {
	output := "git: bad configuration option        \rgit: \n" +
		"git: terminating        \rgit: \n" +
		"remote: failed        \rremote: \n"

	rendered := NewCardRenderer().RenderMessage(
		"jj git push",
		"",
		errors.New(output),
		80,
	)
	plain := ansi.Strip(rendered)

	assert.Equal(t, 2, strings.Count(plain, "git:"))
	assert.Equal(t, 1, strings.Count(plain, "remote:"))
	assert.NotRegexp(t, `(?m)^│ git: +│$`, plain)
	assert.NotRegexp(t, `(?m)^│ remote: +│$`, plain)
}
