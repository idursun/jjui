package dispatch

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/config"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/stretchr/testify/assert"
)

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func makeResolver(bindings []keybindings.Binding, configured map[keybindings.Action]config.ActionConfig) *Resolver {
	d, err := NewDispatcher(bindings)
	if err != nil {
		panic(err)
	}
	if configured == nil {
		configured = make(map[keybindings.Action]config.ActionConfig)
	}
	return NewResolver(d, configured)
}

func TestResolveKey_BuiltInAction(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "ui.quit", Scope: "ui", Key: []string{"q"}},
	}, nil)

	result := r.ResolveKey(keyMsg("q"), []keybindings.Scope{"ui"}, nil)
	assert.True(t, result.Consumed)
	assert.NotNil(t, result.Intent)
	_, isQuit := result.Intent.(intents.Quit)
	assert.True(t, isQuit)
	assert.Equal(t, "ui", result.Owner)
}

func TestResolveKey_Pending(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "ui.quit", Scope: "ui", Seq: []string{"g", "q"}},
	}, nil)

	result := r.ResolveKey(keyMsg("g"), []keybindings.Scope{"ui"}, nil)
	assert.True(t, result.Pending)
	assert.True(t, result.Consumed)
	assert.Nil(t, result.Intent)
}

func TestResolveKey_Unmatched(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "ui.quit", Scope: "ui", Key: []string{"q"}},
	}, nil)

	result := r.ResolveKey(keyMsg("x"), []keybindings.Scope{"ui"}, nil)
	assert.False(t, result.Consumed)
	assert.Nil(t, result.Intent)
}

func TestResolveKey_ConfiguredLuaAction(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "my_action", Scope: "ui", Key: []string{"m"}},
	}, map[keybindings.Action]config.ActionConfig{
		"my_action": {Lua: "print('hello')"},
	})

	result := r.ResolveKey(keyMsg("m"), []keybindings.Scope{"ui"}, nil)
	assert.True(t, result.Consumed)
	assert.Nil(t, result.Intent)
	assert.Equal(t, "print('hello')", result.LuaScript)
}

func TestResolveKey_OverrideTakesPrecedence(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "revisions.move_down", Scope: "revisions", Key: []string{"j"}},
	}, nil)

	override := func(action keybindings.Action, args map[string]any) (intents.Intent, bool) {
		if action == "revisions.move_down" {
			return intents.Navigate{Delta: 42}, true
		}
		return nil, false
	}

	result := r.ResolveKey(keyMsg("j"), []keybindings.Scope{"revisions"}, override)
	assert.True(t, result.Consumed)
	nav, ok := result.Intent.(intents.Navigate)
	assert.True(t, ok)
	assert.Equal(t, 42, nav.Delta)
}

func TestResolveKey_ConfiguredActionWithoutLuaIsNoop(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "bad_action", Scope: "ui", Key: []string{"a"}},
	}, map[keybindings.Action]config.ActionConfig{
		"bad_action": {},
	})

	result := r.ResolveKey(keyMsg("a"), []keybindings.Scope{"ui"}, nil)
	assert.True(t, result.Consumed)
	assert.Nil(t, result.Intent)
}

func TestResolveAction_DirectCall(t *testing.T) {
	r := makeResolver([]keybindings.Binding{
		{Action: "ui.quit", Scope: "ui", Key: []string{"q"}},
	}, nil)

	result := r.ResolveAction("ui", "ui.quit", nil, nil)
	assert.True(t, result.Consumed)
	assert.NotNil(t, result.Intent)
	_, isQuit := result.Intent.(intents.Quit)
	assert.True(t, isQuit)
}

func TestDeriveOwner(t *testing.T) {
	tests := []struct {
		action keybindings.Action
		want   string
	}{
		{"ui.quit", "ui"},
		{"revisions.move_down", "revisions"},
		{"move_down", ""},
		{"preview_toggle", ""},
		{"quick_search_next", ""},
		{"revisions.rebase.apply", "revisions.rebase"},
		{"my_action", ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			got := DeriveOwner(tt.action)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsRevisionsOwner(t *testing.T) {
	assert.True(t, IsRevisionsOwner("revisions"))
	assert.True(t, IsRevisionsOwner("revisions.rebase"))
	assert.True(t, IsRevisionsOwner("revisions.details"))
	assert.False(t, IsRevisionsOwner("ui"))
	assert.False(t, IsRevisionsOwner("bookmarks"))
}
