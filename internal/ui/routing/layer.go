package routing

import (
	tea "charm.land/bubbletea/v2"
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

// RouteIntent walks the scope chain and delivers the intent to the first
// handler that accepts it. The returned bool reports whether any scope
// handled the intent, even when the resulting command is nil. If a
// non-leaking scope blocks the intent, routing stops and reports the
// intent as unhandled.
func RouteIntent(scopes []Scope, intent intents.Intent) (tea.Cmd, bool) {
	for _, scope := range scopes {
		if cmd, handled := scope.Handler.HandleIntent(intent); handled {
			return cmd, true
		}
		if !scope.AllowLeak {
			return nil, false
		}
	}
	return nil, false
}
