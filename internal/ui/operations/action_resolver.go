package operations

import (
	"github.com/idursun/jjui/internal/ui/actionargs"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
)

// ActionIntentResolver allows an operation to own how dispatcher actions map to intents.
// Revisions can delegate action resolution to the active operation instead of hardcoding
// operation-specific action switches.
type ActionIntentResolver interface {
	ResolveAction(action keybindings.Action, args map[string]any) (intents.Intent, bool)
}

func BoolArg(args map[string]any, name string, fallback bool) bool {
	return actionargs.BoolArg(args, name, fallback)
}
