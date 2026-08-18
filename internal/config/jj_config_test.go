package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfigRevsetAliases(t *testing.T) {
	t.Run("valid config parsing", func(t *testing.T) {
		tomlData := `
[revsets]
log = "all()"

[templates]
log = "builtin_log_comfortable"

[revset-aliases]
"@" = "HEAD"
"my_alias" = { definition = "trunk()..", doc = "my_alias doc string" }

[revset-aliases."another_alias"]
definition = "mine()"
doc = "another_alias doc"

[revset-aliases."alias_func(to)"]
definition = "heads(::to)"
doc = "alias_func(to) doc"
`
		config, err := parseConfig(tomlData)
		require.NoError(t, err)
		require.NotNil(t, config)

		assert.Equal(t, "all()", config.Revsets.Log)
		assert.Equal(t, "builtin_log_comfortable", config.Templates.Log)

		assert.Len(t, config.RevsetAliases, 4)
		assert.Equal(t, "HEAD", config.RevsetAliases["@"])
		assert.Equal(t, "trunk()..", config.RevsetAliases["my_alias"])
		assert.Equal(t, "mine()", config.RevsetAliases["another_alias"])
		assert.Equal(t, "heads(::to)", config.RevsetAliases["alias_func(to)"])
	})
}
