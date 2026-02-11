package config

import (
	"fmt"
	"strings"

	"github.com/idursun/jjui/internal/ui/actionmeta"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
)

// StringList allows TOML values to be specified as a string or array of strings.
type StringList []string

func (l *StringList) UnmarshalTOML(value any) error {
	switch v := value.(type) {
	case string:
		*l = StringList{v}
		return nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return fmt.Errorf("expected string in list, got %T", item)
			}
			out = append(out, s)
		}
		*l = StringList(out)
		return nil
	default:
		return fmt.Errorf("expected string or list of strings, got %T", value)
	}
}

type ActionConfig struct {
	Name string         `toml:"name"`
	Desc string         `toml:"desc"`
	Lua  string         `toml:"lua"`
	Args map[string]any `toml:"args"`
}

type BindingConfig struct {
	Action string         `toml:"action"`
	Key    StringList     `toml:"key"`
	Seq    StringList     `toml:"seq"`
	Scope  string         `toml:"scope"`
	Args   map[string]any `toml:"args"`
}

func (c *Config) ValidateBindingsAndActions() error {
	if err := validateActions(c.Actions); err != nil {
		return err
	}
	if err := validateBindings(c.Bindings); err != nil {
		return err
	}
	if err := validateActionReferencesAndArgs(c.Actions, c.Bindings); err != nil {
		return err
	}
	return nil
}

func validateActions(actions []ActionConfig) error {
	seen := map[string]struct{}{}
	for i, action := range actions {
		name := strings.TrimSpace(action.Name)
		if name == "" {
			return fmt.Errorf("actions[%d]: name is required", i)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("actions[%d]: duplicate action name %q", i, name)
		}
		seen[name] = struct{}{}

		if len(action.Args) > 0 {
			return fmt.Errorf("actions[%d]: actions.args is not supported; pass args in binding or lua", i)
		}
		if strings.TrimSpace(action.Lua) == "" {
			return fmt.Errorf("actions[%d]: lua is required", i)
		}
	}
	return nil
}

// BindingsToRuntime converts config bindings to runtime bindings,
// skipping entries with empty scope or action.
func BindingsToRuntime(bindings []BindingConfig) []keybindings.Binding {
	out := make([]keybindings.Binding, 0, len(bindings))
	for _, binding := range bindings {
		scope := keybindings.Scope(strings.TrimSpace(binding.Scope))
		action := keybindings.Action(strings.TrimSpace(binding.Action))
		if scope == "" || action == "" {
			continue
		}
		out = append(out, keybindings.Binding{
			Action: action,
			Scope:  scope,
			Key:    append([]string(nil), binding.Key...),
			Seq:    append([]string(nil), binding.Seq...),
			Args:   keybindings.CloneArgs(binding.Args),
		})
	}
	return out
}

func validateBindings(bindings []BindingConfig) error {
	runtimeBindings := BindingsToRuntime(bindings)
	for i, rb := range runtimeBindings {
		if rb.Action == "" {
			return fmt.Errorf("bindings[%d]: action is required", i)
		}
		if rb.Scope == "" {
			return fmt.Errorf("bindings[%d]: scope is required", i)
		}
	}
	if err := keybindings.ValidateBindings(runtimeBindings); err != nil {
		return fmt.Errorf("bindings: %w", err)
	}
	return nil
}

func validateActionReferencesAndArgs(actions []ActionConfig, bindings []BindingConfig) error {
	custom := make(map[string]ActionConfig, len(actions))
	for _, action := range actions {
		name := strings.TrimSpace(action.Name)
		if name == "" {
			continue
		}
		custom[name] = action
	}

	for i, binding := range bindings {
		actionName := strings.TrimSpace(binding.Action)
		if actionName == "" {
			continue
		}
		if _, ok := custom[actionName]; ok {
			continue
		}
		if err := actionmeta.ValidateBuiltInActionArgs(actionName, binding.Args); err != nil {
			return fmt.Errorf("bindings[%d]: %w", i, err)
		}
	}
	return nil
}
