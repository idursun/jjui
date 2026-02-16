package scripting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uicontext "github.com/idursun/jjui/internal/ui/context"
)

func TestRunScriptRequiresInitializedVM(t *testing.T) {
	ctx := &uicontext.MainContext{}
	_, _, err := RunScript(ctx, `x = 1`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestRunSetupActionRegistersActionAndBinding(t *testing.T) {
	ctx := &uicontext.MainContext{}
	require.NoError(t, InitVM(ctx))
	t.Cleanup(func() {
		CloseVM(ctx)
	})

	source := `
function helper()
  marker = "ok"
end

function setup(config)
  config.action("my-action", function()
    helper()
  end, { desc = "My custom action" })

  config.bind({
    action = "my-action",
    key = "x",
    scope = "revisions",
  })
end
`

	actions, bindings, err := RunSetup(ctx, source)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	require.Len(t, bindings, 1)

	assert.Equal(t, "my-action", actions[0].Name)
	assert.Equal(t, "My custom action", actions[0].Desc)
	assert.True(t, strings.HasPrefix(actions[0].Lua, `__jjui_actions["action_`))
	assert.True(t, strings.HasSuffix(actions[0].Lua, `"]()`))

	assert.Equal(t, "my-action", bindings[0].Action)
	assert.Equal(t, "revisions", bindings[0].Scope)
	assert.Equal(t, []string{"x"}, []string(bindings[0].Key))

	runner, _, err := RunScript(ctx, actions[0].Lua)
	require.NoError(t, err)
	require.NotNil(t, runner)
	assert.True(t, runner.Done())
	assert.Equal(t, "ok", ctx.ScriptVM.GetGlobal("marker").String())
}

func TestRunSetupActionWithInlineBindingOpts(t *testing.T) {
	ctx := &uicontext.MainContext{}
	require.NoError(t, InitVM(ctx))
	t.Cleanup(func() {
		CloseVM(ctx)
	})

	source := `
function setup(config)
  config.action("inline-action", function()
    marker = "inline"
  end, {
    desc = "Inline binding action",
    key = "z",
    scope = "revisions",
  })
end
`

	actions, bindings, err := RunSetup(ctx, source)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	require.Len(t, bindings, 1)

	assert.Equal(t, "inline-action", actions[0].Name)
	assert.Equal(t, "Inline binding action", actions[0].Desc)
	assert.Equal(t, "inline-action", bindings[0].Action)
	assert.Equal(t, "revisions", bindings[0].Scope)
	assert.Equal(t, []string{"z"}, []string(bindings[0].Key))
	assert.Empty(t, bindings[0].Seq)

	runner, _, err := RunScript(ctx, actions[0].Lua)
	require.NoError(t, err)
	require.NotNil(t, runner)
	assert.True(t, runner.Done())
	assert.Equal(t, "inline", ctx.ScriptVM.GetGlobal("marker").String())
}

func TestRunSetupActionInlineBindingRequiresScope(t *testing.T) {
	ctx := &uicontext.MainContext{}
	require.NoError(t, InitVM(ctx))
	t.Cleanup(func() {
		CloseVM(ctx)
	})

	source := `
function setup(config)
  config.action("broken", function() end, { key = "z" })
end
`

	_, _, err := RunSetup(ctx, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opts.scope is required")
}

func TestRunSetupCanRequirePluginFromConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("JJUI_CONFIG_DIR", configDir)

	pluginsDir := filepath.Join(configDir, "plugins")
	require.NoError(t, os.MkdirAll(pluginsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pluginsDir, "my_plugin.lua"), []byte(`
local M = {}

function M.setup(config)
  config.action("plugin-action", function()
    marker = "plugin-ok"
  end, {
    desc = "Plugin action",
    key = "P",
    scope = "revisions",
  })
end

return M
`), 0o644))

	ctx := &uicontext.MainContext{}
	require.NoError(t, InitVM(ctx))
	t.Cleanup(func() {
		CloseVM(ctx)
	})

	source := `
local plugin = require("plugins.my_plugin")

function setup(config)
  plugin.setup(config)
end
`

	actions, bindings, err := RunSetup(ctx, source)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	require.Len(t, bindings, 1)
	assert.Equal(t, "plugin-action", actions[0].Name)
	assert.Equal(t, "revisions", bindings[0].Scope)
	assert.Equal(t, []string{"P"}, []string(bindings[0].Key))

	runner, _, err := RunScript(ctx, actions[0].Lua)
	require.NoError(t, err)
	require.NotNil(t, runner)
	assert.True(t, runner.Done())
	assert.Equal(t, "plugin-ok", ctx.ScriptVM.GetGlobal("marker").String())
}
