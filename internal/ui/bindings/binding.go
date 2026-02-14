package bindings

import (
	"fmt"
)

// Binding maps a key (or key sequence) to an action in a scope.
type Binding struct {
	Action Action
	Scope  Scope
	Key    []string
	Seq    []string
	Args   map[string]any
}

func (b Binding) validate() error {
	if b.Action == "" {
		return fmt.Errorf("binding action is required")
	}
	if b.Scope == "" {
		return fmt.Errorf("binding scope is required")
	}

	hasKey := len(b.Key) > 0
	hasSeq := len(b.Seq) > 0
	if hasKey == hasSeq {
		return fmt.Errorf("binding %q in scope %q must set exactly one of key or seq", b.Action, b.Scope)
	}

	for _, key := range b.Key {
		if key == "" {
			return fmt.Errorf("binding %q in scope %q contains empty key", b.Action, b.Scope)
		}
	}
	for _, seqKey := range b.Seq {
		if seqKey == "" {
			return fmt.Errorf("binding %q in scope %q contains empty sequence key", b.Action, b.Scope)
		}
	}
	if len(b.Seq) == 1 {
		return fmt.Errorf("binding %q in scope %q has seq with only one key; use key instead", b.Action, b.Scope)
	}

	return nil
}

func ValidateBindings(bindings []Binding) error {
	for i, binding := range bindings {
		if err := binding.validate(); err != nil {
			return fmt.Errorf("invalid binding at index %d: %w", i, err)
		}
	}
	return nil
}
