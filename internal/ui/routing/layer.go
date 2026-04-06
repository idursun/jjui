package routing

import (
	tea "charm.land/bubbletea/v2"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
)

// Layer represents one routing layer in the intent dispatch chain.
// Layers are ordered from innermost (highest priority) to outermost.
type Layer struct {
	Scope keybindings.ScopeName

	AllowLeak bool

	Handler LayerHandler
}

type LayerHandler interface {
	HandleIntent(intent intents.Intent) (tea.Cmd, bool)
	Update(msg tea.Msg) tea.Cmd
}

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
