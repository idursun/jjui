package dispatch

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/actionmeta"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
)

// Scope represents one routing layer in the intent dispatch chain.
// Scopes are ordered from innermost (highest priority) to outermost.
type Scope struct {
	Name      keybindings.ScopeName
	AllowLeak bool
	Handler   ScopeHandler
}

type ScopeHandler interface {
	HandleIntent(intent intents.Intent) (tea.Cmd, bool)
	Update(msg tea.Msg) tea.Cmd
}

type ScopeProvider interface {
	Scopes() []Scope
}

func VisibleScopes(scopes []Scope) []Scope {
	for i, scope := range scopes {
		if !scope.AllowLeak {
			return scopes[:i+1]
		}
	}
	return scopes
}

func RouteIntent(scopes []Scope, intent intents.Intent) (tea.Cmd, bool) {
	for _, scope := range VisibleScopes(scopes) {
		if cmd, handled := scope.Handler.HandleIntent(intent); handled {
			return cmd, true
		}
	}
	return nil, false
}

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
