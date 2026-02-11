package bindings

import "fmt"

// Registry stores the known action names.
type Registry struct {
	actions map[Action]struct{}
}

func NewRegistry(actions []Action) *Registry {
	registry := &Registry{actions: make(map[Action]struct{}, len(actions))}
	for _, action := range actions {
		if action == "" {
			continue
		}
		registry.actions[action] = struct{}{}
	}
	return registry
}

func (r *Registry) Has(action Action) bool {
	if r == nil {
		return false
	}
	_, ok := r.actions[action]
	return ok
}

func (r *Registry) ValidateAction(action Action) error {
	if !r.Has(action) {
		return fmt.Errorf("unknown action %q", action)
	}
	return nil
}
