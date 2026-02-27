package revisions

import (
	tea "charm.land/bubbletea/v2"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/operations"
)

func (m *Model) HandleDispatchedAction(action keybindings.Action, args map[string]any) (tea.Cmd, bool) {
	if resolver, ok := m.CurrentOperation().(operations.ActionIntentResolver); ok {
		if intent, resolved := resolver.ResolveAction(action, args); resolved {
			return m.CurrentOperation().Update(intent), true
		}
	}

	if intent, ok := actions.ResolveByAction(action, args); ok {
		return m.Update(intent), true
	}
	return nil, false
}
