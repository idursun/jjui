package actions

import (
	"strings"

	"github.com/idursun/jjui/internal/ui/actionmeta"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
)

// ResolveByAction resolves an action to an intent without requiring callers
// to pass an owner. Action ownership is discovered from generated action metadata.
func ResolveByAction(action keybindings.Action, args map[string]any) (intents.Intent, bool) {
	name := strings.TrimSpace(string(action))
	if name == "" {
		return nil, false
	}
	for _, owner := range actionmeta.ActionOwners(name) {
		if intent, ok := ResolveIntent(owner, action, args); ok {
			return intent, true
		}
	}
	return nil, false
}

// ResolveByScopeStrict resolves only within the given scope owner and does not
// fall back to broader owners.
func ResolveByScopeStrict(scope keybindings.Scope, action keybindings.Action, args map[string]any) (intents.Intent, bool) {
	scopeName := strings.TrimSpace(string(scope))
	if scopeName == "" {
		return ResolveByAction(action, args)
	}
	actionName := strings.TrimSpace(string(action))
	if actionName == "" {
		return nil, false
	}
	for _, owner := range actionmeta.ActionOwners(actionName) {
		if scopeAllowsOwner(scopeName, owner) {
			return ResolveIntent(owner, action, args)
		}
	}
	return nil, false
}

func scopeAllowsOwner(scope string, owner string) bool {
	if owner == "" {
		return false
	}
	return scope == owner || strings.HasPrefix(scope, owner+".")
}
