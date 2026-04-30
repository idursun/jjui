package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ActionsAndBindings_NewSchema(t *testing.T) {
	content := `
[[actions]]
name = "apply_parallel"
desc = "Apply in parallel"
lua = "print('apply')"

[[bindings]]
action = "apply_parallel"
scope = "revisions.squash"
key = ["alt+shift+enter"]

[[bindings]]
action = "ui.open_git"
scope = "revisions"
seq = ["g", "g"]
`

	cfg := &Config{}
	err := cfg.Load(content, "")
	require.NoError(t, err)
	require.Len(t, cfg.Actions, 1)
	require.Len(t, cfg.Bindings, 2)

	assert.Equal(t, "apply_parallel", cfg.Actions[0].Name)
	assert.Equal(t, "print('apply')", cfg.Actions[0].Lua)
	assert.Equal(t, StringList{"alt+shift+enter"}, cfg.Bindings[0].Key)
	assert.Equal(t, StringList{"g", "g"}, cfg.Bindings[1].Seq)
}

func TestLoad_ActionCanDefineBinding(t *testing.T) {
	content := `
[[actions]]
name = "apply_parallel"
desc = "Apply in parallel"
lua = "print('apply')"
scope = "revisions.squash"
key = "alt+shift+enter"

[[actions]]
name = "open_git_alias"
lua = "print('git')"
scope = "revisions"
seq = ["g", "g"]
`

	cfg := &Config{}
	err := cfg.Load(content, "")
	require.NoError(t, err)
	require.Len(t, cfg.Actions, 2)
	require.Len(t, cfg.Bindings, 2)

	assert.Equal(t, ActionConfig{Name: "apply_parallel", Lua: "print('apply')"}, cfg.Actions[0])
	assert.Equal(t, BindingConfig{
		Action: "apply_parallel",
		Desc:   "Apply in parallel",
		Scope:  "revisions.squash",
		Key:    StringList{"alt+shift+enter"},
	}, cfg.Bindings[0])
	assert.Equal(t, BindingConfig{
		Action: "open_git_alias",
		Scope:  "revisions",
		Seq:    StringList{"g", "g"},
	}, cfg.Bindings[1])
}

func TestLoad_ActionDefinedBindingComposesWithExplicitBindings(t *testing.T) {
	content := `
[[actions]]
name = "custom"
lua = "print('custom')"
scope = "revisions"
key = "x"

[[bindings]]
action = "custom"
scope = "git"
key = "x"
`

	cfg := &Config{}
	err := cfg.Load(content, "")
	require.NoError(t, err)
	require.Len(t, cfg.Bindings, 2)

	assert.Equal(t, BindingConfig{Action: "custom", Scope: "revisions", Key: StringList{"x"}}, cfg.Bindings[0])
	assert.Equal(t, BindingConfig{Action: "custom", Scope: "git", Key: StringList{"x"}}, cfg.Bindings[1])
}

func TestLoad_ActionsAndBindings_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "action invalid action args",
			content: `
[[actions]]
name = "bad_apply"
lua = "print('x')"
args = { force = true }
`,
			want: "actions.args is not supported",
		},
		{
			name: "action missing lua",
			content: `
[[actions]]
name = "my_action"
`,
			want: "lua is required",
		},
		{
			name: "action binding missing scope",
			content: `
[[actions]]
name = "my_action"
lua = "print('x')"
key = "x"
`,
			want: "scope is required when key or seq is set",
		},
		{
			name: "action binding invalid key and seq together",
			content: `
[[actions]]
name = "my_action"
lua = "print('x')"
scope = "revisions"
key = "x"
seq = ["g", "x"]
`,
			want: "must set exactly one of key or seq",
		},
		{
			name: "binding invalid key and seq together",
			content: `
[[bindings]]
action = "ui.open_git"
scope = "revisions"
key = ["g"]
seq = ["g", "f"]
`,
			want: "must set exactly one of key or seq",
		},
		{
			name: "binding unknown action",
			content: `
[[bindings]]
action = "does_not_exist"
scope = "revisions"
key = ["x"]
`,
			want: "unknown built-in action",
		},
		{
			name: "binding invalid built_in args type",
			content: `
[[bindings]]
action = "revisions.squash.apply"
scope = "revisions.squash"
key = ["enter"]
args = { force = "yes" }
`,
			want: "expects bool",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{}
			err := cfg.Load(tc.content, "")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestLoad_CanonicalBuiltInEnumArgsValidation(t *testing.T) {
	valid := `
[[bindings]]
action = "revisions.revert.set_target"
scope = "revisions.revert"
key = ["b"]
args = { target = "before" }
`
	cfg := &Config{}
	require.NoError(t, cfg.Load(valid, ""))

	invalid := `
[[bindings]]
action = "revisions.revert.set_target"
scope = "revisions.revert"
key = ["b"]
`
	cfg = &Config{}
	err := cfg.Load(invalid, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires arg")
}
