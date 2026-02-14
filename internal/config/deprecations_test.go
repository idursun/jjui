package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeprecatedConfigWarnings(t *testing.T) {
	content := `
[keys]
up = ["k"]

[custom_commands]
"x" = { args = ["status"] }

[leader]
"gr" = { send = ["git", "remote"] }
`
	warnings := DeprecatedConfigWarnings(content)
	assert.Len(t, warnings, 2)
	assert.Contains(t, warnings[0], "[custom_commands]")
	assert.Contains(t, warnings[1], "[leader]")
}
