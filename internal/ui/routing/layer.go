package routing

import (
	tea "charm.land/bubbletea/v2"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
)

// Layer represents one routing layer in the intent dispatch chain.
// Layers are ordered from innermost (highest priority) to outermost.
type Layer struct {
	// Scope lists the binding scopes this layer contributes, from most
	// specific to least specific. Used for key->action resolution.
	// Example: []Scope{"revisions.details", "revisions"}
	Scope keybindings.Scope

	// AllowLeak controls whether unhandled intents may continue to the
	// next outer layer. Default should be false.
	// true  = transparent (revisions in normal mode, ui.preview)
	// false = blocking (text inputs, confirmation dialogs, diff, stacked modals)
	AllowLeak bool

	// Handler receives resolved intents and unmatched messages for this layer.
	Handler LayerHandler
}

// LayerHandler is implemented by any model that can receive intents
// through the layer routing system and raw unmatched messages.
type LayerHandler interface {
	HandleIntent(intent intents.Intent) (tea.Cmd, bool)
	Update(msg tea.Msg) tea.Cmd
}

// LayerProvider is implemented by models that contribute layers to the
// dispatch chain. ui.Model calls Layers() unconditionally on all children.
// Models return nil when they are inactive (not editing, not focused, not visible).
type LayerProvider interface {
	Layers() []Layer
}

// RouteIntent walks the layer chain and delivers the intent to the first
// handler that accepts it. The returned bool reports whether any layer
// handled the intent, even when the resulting command is nil. If a
// non-leaking layer blocks the intent, routing stops and reports the
// intent as unhandled.
func RouteIntent(layers []Layer, intent intents.Intent) (tea.Cmd, bool) {
	for _, layer := range layers {
		if cmd, handled := layer.Handler.HandleIntent(intent); handled {
			return cmd, true
		}
		if !layer.AllowLeak {
			return nil, false
		}
	}
	return nil, false
}
