package dispatch

import (
	"strings"

	"github.com/idursun/jjui/internal/ui/actionmeta"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
)

// DeriveScope determines the intent scope from generated built-in metadata.
// Non-built-in actions have no scope.
func DeriveScope(action keybindings.Action) string {
	actionName := strings.TrimSpace(string(action))
	if actionName == "" {
		return ""
	}
	if scopes := actionmeta.ActionScopes(actionName); len(scopes) > 0 {
		return scopes[0]
	}
	return ""
}
