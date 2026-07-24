package common

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/actionmeta"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
)

// LeakPolicy controls which outer scopes remain visible after a scope is reached.
type LeakPolicy int

const (
	// LeakAll allows routing to continue through every outer scope.
	LeakAll LeakPolicy = iota
	// LeakGlobal allows routing to continue only through outer scopes marked Global.
	LeakGlobal
	// LeakNone prevents routing to any outer scope.
	LeakNone
)

// Scope represents one routing layer in the intent dispatch chain.
// Scopes are ordered from innermost (highest priority) to outermost.
type Scope struct {
	Name    keybindings.ScopeName
	Leak    LeakPolicy
	Global  bool
	Handler ScopeHandler
}

type ScopeHandler interface {
	// HandleIntent returns handled=true when this scope consumes the intent, even
	// if it has no command to return. Returning false allows the next visible
	// outer scope to handle the intent.
	HandleIntent(intent intents.Intent) (tea.Cmd, bool)
	Update(msg tea.Msg) tea.Cmd
}

type ScopeProvider interface {
	Scopes() []Scope
}

func VisibleScopes(scopes []Scope) []Scope {
	for i, scope := range scopes {
		if scope.Leak == LeakNone {
			return scopes[:i+1]
		}
		if scope.Leak == LeakGlobal {
			result := make([]Scope, i+1)
			copy(result, scopes[:i+1])
			for _, s := range scopes[i+1:] {
				if s.Global {
					result = append(result, s)
				}
			}
			return result
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
