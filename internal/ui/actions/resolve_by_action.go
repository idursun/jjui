package actions

import (
	"strings"

	"github.com/idursun/jjui/internal/ui/actionmeta"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
)

// ResolveByAction resolves an action to an intent without requiring callers
// to pass a scope. Action scope membership is discovered from generated action metadata.
func ResolveByAction(action keybindings.Action, args map[string]any) (intents.Intent, bool) {
	name := strings.TrimSpace(string(action))
	if name == "" {
		return nil, false
	}
	for _, scope := range actionmeta.ActionScopes(name) {
		if intent, ok := ResolveIntent(scope, action, args); ok {
			return intent, true
		}
	}
	return nil, false
}
