package actions

import (
	"testing"

	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/stretchr/testify/require"
)

func TestResolveIntent_ApplyForceFromArgs(t *testing.T) {
	intent, ok := ResolveIntent(ScopeSquash, keybindings.Action("revisions.squash.apply"), map[string]any{"force": true})
	require.True(t, ok)
	apply, ok := intent.(intents.Apply)
	require.True(t, ok)
	require.True(t, apply.Force)
}

func TestResolveIntent_TargetPickerApplyForceFromArgs(t *testing.T) {
	intent, ok := ResolveIntent(ScopeTargetPicker, keybindings.Action("revisions.target_picker.apply"), map[string]any{"force": true})
	require.True(t, ok)
	apply, ok := intent.(intents.TargetPickerApply)
	require.True(t, ok)
	require.True(t, apply.Force)
}

func TestResolveIntent_UnknownScopeOrAction(t *testing.T) {
	_, ok := ResolveIntent("unknown.scope", keybindings.Action("revisions.apply"), nil)
	require.False(t, ok)

	_, ok = ResolveIntent(ScopeSquash, keybindings.Action("ui.open_git"), nil)
	require.False(t, ok)
}
