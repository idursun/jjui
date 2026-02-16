package scripting

import (
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
