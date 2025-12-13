package bindings

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

// KeyBinding represents a single keybinding entry loaded from configuration.
// Disabled entries keep their shape but won't be dispatched.
type KeyBinding struct {
	Keys        []string
	KeySequence []string `toml:"key_sequence"`
	Action      string
	When        string
	Args        map[string]any
	Disabled    bool
	Condition   Condition
}

// Load parses [[keybindings]] entries from the given TOML content.
// A leading "-" on the action name marks the binding as disabled.
func Load(data string) ([]KeyBinding, error) {
	type bindingsTOML struct {
		KeyBindings []struct {
			Keys        []string               `toml:"keys"`
			KeySequence []string               `toml:"key_sequence"`
			Action      string                 `toml:"action"`
			When        string                 `toml:"when"`
			Args        map[string]interface{} `toml:"args"`
		} `toml:"keybindings"`
	}

	var file bindingsTOML
	if _, err := toml.Decode(data, &file); err != nil {
		return nil, err
	}

	bindings := make([]KeyBinding, 0, len(file.KeyBindings))
	for i, b := range file.KeyBindings {
		if len(b.Keys) == 0 && len(b.KeySequence) == 0 {
			return nil, fmt.Errorf("keybindings[%d]: keys or key_sequence cannot be empty", i)
		}
		if len(b.Keys) > 0 && len(b.KeySequence) > 0 {
			return nil, fmt.Errorf("keybindings[%d]: keys and key_sequence cannot both be set", i)
		}
		when := strings.TrimSpace(b.When)
		kb := KeyBinding{
			Keys:        b.Keys,
			KeySequence: b.KeySequence,
			When:        when,
			Args:        b.Args,
		}
		action := strings.TrimSpace(b.Action)
		if strings.HasPrefix(action, "-") {
			kb.Disabled = true
			action = strings.TrimPrefix(action, "-")
			action = strings.TrimSpace(action)
		}
		if action == "" {
			return nil, fmt.Errorf("keybindings[%d]: action cannot be empty", i)
		}
		kb.Action = action
		if when != "" {
			cond, err := ParseCondition(when)
			if err != nil {
				return nil, fmt.Errorf("keybindings[%d]: invalid when clause: %w", i, err)
			}
			kb.Condition = cond
		}
		bindings = append(bindings, kb)
	}
	return bindings, nil
}
