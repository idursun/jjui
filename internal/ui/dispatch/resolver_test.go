package dispatch

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/stretchr/testify/assert"
)

func keyMsg(s string) tea.KeyPressMsg {
	runes := []rune(s)
	code := rune(0)
	if len(runes) > 0 {
		code = runes[0]
	}
	return tea.KeyPressMsg{Text: s, Code: code}
}

func makeResolver(bindings []keybindings.Binding, configured map[keybindings.Action]config.ActionConfig) *Resolver {
	d, err := NewDispatcher(bindings)
	if err != nil {
		panic(err)
	}
	actions := make([]config.ActionConfig, 0, len(configured))
	for name, action := range configured {
		action.Name = string(name)
		actions = append(actions, action)
	}
	return newResolverWithActions(d, actions)
}

func TestResolveKey_BuiltInAction(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "ui.quit", Scope: "ui", Key: []string{"q"}},
	}, nil)

	result := r.ResolveKey(keyMsg("q"), createScopes("ui"))
	assert.True(t, result.Consumed)
	assert.NotNil(t, result.Intent)
	_, isQuit := result.Intent.(intents.Quit)
	assert.True(t, isQuit)
	assert.Equal(t, "ui", result.Scope)
}

func TestResolveKey_Pending(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "ui.quit", Scope: "ui", Seq: []string{"g", "q"}},
	}, nil)

	result := r.ResolveKey(keyMsg("g"), createScopes("ui"))
	assert.True(t, result.Pending)
	assert.True(t, result.Consumed)
	assert.Nil(t, result.Intent)
}

func TestResolveKey_Unmatched(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "ui.quit", Scope: "ui", Key: []string{"q"}},
	}, nil)

	result := r.ResolveKey(keyMsg("x"), createScopes("ui"))
	assert.False(t, result.Consumed)
	assert.Nil(t, result.Intent)
}

func TestResolveKey_ConfiguredLuaAction(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "my_action", Scope: "ui", Key: []string{"m"}},
	}, map[keybindings.Action]config.ActionConfig{
		"my_action": {Lua: "print('hello')"},
	})

	result := r.ResolveKey(keyMsg("m"), createScopes("ui"))
	assert.True(t, result.Consumed)
	assert.Nil(t, result.Intent)
	assert.Equal(t, "print('hello')", result.LuaScript)
}

func TestResolveKey_BuiltInCatalogResolution(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "revisions.move_down", Scope: "revisions", Key: []string{"j"}},
	}, nil)

	result := r.ResolveKey(keyMsg("j"), createScopes("revisions"))
	assert.True(t, result.Consumed)
	nav, ok := result.Intent.(intents.Navigate)
	assert.True(t, ok)
	assert.Equal(t, 1, nav.Delta)
}

func TestResolveKey_UsesMatchedBindingScopeForRouting(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "revset.edit", Scope: "revisions", Key: []string{"L"}},
	}, nil)

	result := r.ResolveKey(keyMsg("L"), createScopes("revisions", "ui"))
	assert.True(t, result.Consumed)
	_, ok := result.Intent.(intents.Edit)
	assert.True(t, ok)
	assert.Equal(t, "revisions", result.Scope)
}

func TestResolveAction_ConfiguredLuaTakesPrecedenceOverBuiltIn(t *testing.T) {
	r := makeResolver(nil, map[keybindings.Action]config.ActionConfig{
		"revisions.details.diff": {Lua: "flash('override')"},
	})

	result := r.ResolveAction("revisions.details.diff", nil)
	assert.True(t, result.Consumed)
	assert.Nil(t, result.Intent)
	assert.Equal(t, "flash('override')", result.LuaScript)
}

func TestResolveKey_ConfiguredActionWithoutLuaIsNoop(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "bad_action", Scope: "ui", Key: []string{"a"}},
	}, map[keybindings.Action]config.ActionConfig{
		"bad_action": {},
	})

	result := r.ResolveKey(keyMsg("a"), createScopes("ui"))
	assert.True(t, result.Consumed)
	assert.Nil(t, result.Intent)
}

func TestResolveAction_DirectCall(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "ui.quit", Scope: "ui", Key: []string{"q"}},
	}, nil)

	result := r.ResolveAction("ui.quit", nil)
	assert.True(t, result.Consumed)
	assert.NotNil(t, result.Intent)
	_, isQuit := result.Intent.(intents.Quit)
	assert.True(t, isQuit)
}

func TestResolveBuiltInAction_IgnoresConfiguredLuaOverride(t *testing.T) {
	r := makeResolver(nil, map[keybindings.Action]config.ActionConfig{
		"ui.quit": {Lua: "flash('override')"},
	})

	result := r.ResolveBuiltInAction("ui.quit", nil)
	assert.True(t, result.Consumed)
	assert.Empty(t, result.LuaScript)
	assert.NotNil(t, result.Intent)
	_, isQuit := result.Intent.(intents.Quit)
	assert.True(t, isQuit)
}

func TestDeriveScope(t *testing.T) {
	tests := []struct {
		action keybindings.Action
		want   string
	}{
		{"ui.quit", "ui"},
		{"revisions.move_down", "revisions"},
		{"move_down", ""},
		{"preview_toggle", ""},
		{"revisions.quick_search.next", "revisions.quick_search"},
		{"revisions.rebase.apply", "revisions.rebase"},
		{"my_action", ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			got := DeriveScope(tt.action)
			assert.Equal(t, tt.want, got)
		})
	}
}
