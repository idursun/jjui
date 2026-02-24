package scripting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/idursun/jjui/internal/config"
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

func TestRunSetupRegistersAndRunsAction(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		actionName   string
		bindingScope string
		bindingDesc  string
		bindingKey   []string
		marker       string
	}{
		{
			name: "action with config.bind",
			source: `
function helper()
  marker = "ok"
end

function setup(config)
  config.action("my-action", function()
    helper()
  end)

  config.bind({
    action = "my-action",
    desc = "My custom action",
    key = "x",
    scope = "revisions",
  })
end
`,
			actionName:   "my-action",
			bindingScope: "revisions",
			bindingDesc:  "My custom action",
			bindingKey:   []string{"x"},
			marker:       "ok",
		},
		{
			name: "action with inline binding opts",
			source: `
function setup(config)
  config.action("inline-action", function()
    marker = "inline"
  end, {
    desc = "Inline binding action",
    key = "z",
    scope = "revisions",
  })
end
`,
			actionName:   "inline-action",
			bindingScope: "revisions",
			bindingDesc:  "Inline binding action",
			bindingKey:   []string{"z"},
			marker:       "inline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := setupVM(t)
			cfg := *config.Current

			err := RunSetup(ctx, &cfg, tt.source)
			require.NoError(t, err)

			action, ok := findActionByName(cfg.Actions, tt.actionName)
			require.True(t, ok)
			assert.True(t, strings.HasPrefix(action.Lua, `__jjui_actions["action_`))
			assert.True(t, strings.HasSuffix(action.Lua, `"]()`))

			binding, ok := findBinding(cfg.Bindings, tt.actionName, tt.bindingScope)
			require.True(t, ok)
			assert.Equal(t, tt.bindingDesc, binding.Desc)
			assert.Equal(t, tt.bindingKey, []string(binding.Key))

			runner, _, err := RunScript(ctx, action.Lua)
			require.NoError(t, err)
			require.NotNil(t, runner)
			assert.True(t, runner.Done())
			assert.Equal(t, tt.marker, ctx.ScriptVM.GetGlobal("marker").String())
		})
	}
}

func TestRunSetupActionInlineBindingRequiresScope(t *testing.T) {
	ctx := setupVM(t)
	cfg := *config.Current

	source := `
function setup(config)
  config.action("broken", function() end, { key = "z" })
end
`

	err := RunSetup(ctx, &cfg, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opts.scope is required")
}

func TestRunSetupCanRequirePluginFromConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("JJUI_CONFIG_DIR", configDir)
	cfg := *config.Current

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

	ctx := setupVM(t)

	source := `
local plugin = require("plugins.my_plugin")

function setup(config)
  plugin.setup(config)
end
`

	err := RunSetup(ctx, &cfg, source)
	require.NoError(t, err)

	action, ok := findActionByName(cfg.Actions, "plugin-action")
	require.True(t, ok)
	binding, ok := findBinding(cfg.Bindings, "plugin-action", "revisions")
	require.True(t, ok)
	assert.Equal(t, []string{"P"}, []string(binding.Key))

	runner, _, err := RunScript(ctx, action.Lua)
	require.NoError(t, err)
	require.NotNil(t, runner)
	assert.True(t, runner.Done())
	assert.Equal(t, "plugin-ok", ctx.ScriptVM.GetGlobal("marker").String())
}

func TestRunSetupUpdatesConfigAndPreservesExplicitFalse(t *testing.T) {
	ctx := setupVM(t)
	cfg := *config.Current

	source := `
function setup(config)
  config.limit = 5
  config.ui.colors.selected = { bg = "0", underline = false }
end
`

	err := RunSetup(ctx, &cfg, source)
	require.NoError(t, err)

	assert.Equal(t, 5, cfg.Limit)
	selected, ok := cfg.UI.Colors["selected"]
	require.True(t, ok)
	assert.Equal(t, "0", selected.Bg)
	if assert.NotNil(t, selected.Underline) {
		assert.False(t, *selected.Underline)
	}
}

func TestRunSetupExposesRuntimeTerminalAndRepoFields(t *testing.T) {
	ctx := setupVM(t)
	ctx.Location = "/tmp/jjui-test-repo"
	cfg := *config.Current

	source := `
function setup(config)
  marker_repo = config.repo
  marker_dark_mode_type = type(config.terminal.dark_mode)
  marker_bg_type = type(config.terminal.bg)
  marker_fg_type = type(config.terminal.fg)
end
`

	err := RunSetup(ctx, &cfg, source)
	require.NoError(t, err)

	assert.Equal(t, "/tmp/jjui-test-repo", ctx.ScriptVM.GetGlobal("marker_repo").String())
	assert.Equal(t, "boolean", ctx.ScriptVM.GetGlobal("marker_dark_mode_type").String())
	assert.Equal(t, "string", ctx.ScriptVM.GetGlobal("marker_bg_type").String())
	assert.Equal(t, "string", ctx.ScriptVM.GetGlobal("marker_fg_type").String())
}

func TestRunSetupAppliesActionsAndBindingsAssignments(t *testing.T) {
	ctx := setupVM(t)
	cfg := *config.Current

	source := `
function setup(config)
  config.actions = { { name = "replaced", lua = "flash('ok')" } }
  config.bindings = { { action = "replaced", scope = "revisions", key = {"x"} } }
  config.limit = 7
end
`

	err := RunSetup(ctx, &cfg, source)
	require.NoError(t, err)

	assert.Equal(t, 7, cfg.Limit)
	require.Len(t, cfg.Actions, 1)
	assert.Equal(t, "replaced", cfg.Actions[0].Name)
	require.Len(t, cfg.Bindings, 1)
	assert.Equal(t, "replaced", cfg.Bindings[0].Action)
	assert.Equal(t, []string{"x"}, []string(cfg.Bindings[0].Key))
}

func TestRunSetupReportsConfigTypeErrors(t *testing.T) {
	ctx := setupVM(t)
	cfg := *config.Current

	source := `
function setup(config)
  config.limit = "oops"
end
`

	err := RunSetup(ctx, &cfg, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config.limit: expected integer, got string")
}

func TestRunSetupValidatesResultingBindings(t *testing.T) {
	ctx := setupVM(t)
	cfg := *config.Current

	source := `
function setup(config)
  config.bindings = {
    {
      action = "ui.quit",
      scope = "ui",
      key = {"q"},
      seq = {"g", "q"},
    },
  }
end
`

	err := RunSetup(ctx, &cfg, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must set exactly one of key or seq")
}

func TestRunSetupBindShadowsDefaultBindingForSameKey(t *testing.T) {
	ctx := setupVM(t)
	cfg := config.Config{
		Bindings: []config.BindingConfig{
			{Scope: "revisions", Action: "revisions.diff", Key: config.StringList{"d"}},
		},
	}

	source := `
function setup(config)
  config.action("show-diff-in-diffnav", function() end)
  config.bind({ action = "show-diff-in-diffnav", scope = "revisions", key = "d" })
end
`

	err := RunSetup(ctx, &cfg, source)
	require.NoError(t, err)

	_, actionExists := findActionByName(cfg.Actions, "show-diff-in-diffnav")
	assert.True(t, actionExists)
	binding, ok := findBinding(cfg.Bindings, "show-diff-in-diffnav", "revisions")
	require.True(t, ok)
	assert.Equal(t, []string{"d"}, []string(binding.Key))

}

func TestRunSetupActionLastDefinitionWins(t *testing.T) {
	ctx := setupVM(t)
	cfg := *config.Current

	source := `
function setup(config)
  config.action("duplicate-action", function()
    marker = "first"
  end)

  config.action("duplicate-action", function()
    marker = "second"
  end)
end
`

	err := RunSetup(ctx, &cfg, source)
	require.NoError(t, err)

	action, ok := findActionByName(cfg.Actions, "duplicate-action")
	require.True(t, ok)
	runner, _, err := RunScript(ctx, action.Lua)
	require.NoError(t, err)
	require.NotNil(t, runner)
	assert.True(t, runner.Done())
	assert.Equal(t, "second", ctx.ScriptVM.GetGlobal("marker").String())
}

func setupVM(t *testing.T) *uicontext.MainContext {
	t.Helper()
	ctx := &uicontext.MainContext{}
	require.NoError(t, InitVM(ctx))
	t.Cleanup(func() {
		CloseVM(ctx)
	})
	return ctx
}

func findActionByName(actions []config.ActionConfig, name string) (config.ActionConfig, bool) {
	for i := len(actions) - 1; i >= 0; i-- {
		if actions[i].Name == name {
			return actions[i], true
		}
	}
	return config.ActionConfig{}, false
}

func findBinding(bindings []config.BindingConfig, action, scope string) (config.BindingConfig, bool) {
	for _, binding := range bindings {
		if binding.Action == action && binding.Scope == scope {
			return binding, true
		}
	}
	return config.BindingConfig{}, false
}
